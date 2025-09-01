package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	middleware "github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/http/midleware"
)

/* ---------- helpers ---------- */

func signWithClaims(t *testing.T, secret string, claims jwt.MapClaims) string {
	t.Helper()
	// default exp se não vier
	if _, ok := claims["exp"]; !ok {
		claims["exp"] = time.Now().Add(time.Hour).Unix()
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	s, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func newRouterWithJWT(secret string, next gin.HandlerFunc, roles ...string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	g := r.Group("/api")
	g.Use(middleware.JWT(secret))
	if len(roles) > 0 {
		g.Use(middleware.RequireRole(roles...))
	}
	g.GET("/ping", next)
	return r
}

/* ---------- tests JWT ---------- */

func TestJWT_MissingToken(t *testing.T) {
	r := newRouterWithJWT("secret", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d want 401", w.Code)
	}
}

func TestJWT_InvalidSignature(t *testing.T) {
	// token assinado com outro segredo
	badToken := signWithClaims(t, "bad", jwt.MapClaims{"uid": 1, "role": "admin"})
	r := newRouterWithJWT("good", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("Authorization", "Bearer "+badToken)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d want 401", w.Code)
	}
}

func TestJWT_MissingUID(t *testing.T) {
	token := signWithClaims(t, "s", jwt.MapClaims{"role": "admin"})
	r := newRouterWithJWT("s", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("got %d want 401", w.Code)
	}
}

func TestJWT_UIDFloat64AndRoleString_OK(t *testing.T) {
	token := signWithClaims(t, "s", jwt.MapClaims{"uid": float64(42), "role": "editor"})
	var gotUID any
	var gotRole any
	r := newRouterWithJWT("s", func(c *gin.Context) {
		gotUID, _ = c.Get("userID")
		gotRole, _ = c.Get("role")
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d want 200", w.Code)
	}
	if gotUID == nil || gotUID.(uint) != 42 {
		t.Fatalf("userID esperado 42, got %#v", gotUID)
	}
	if gotRole == nil || gotRole.(string) != "editor" {
		t.Fatalf("role esperado editor, got %#v", gotRole)
	}
}

func TestJWT_UIDString_OK(t *testing.T) {
	token := signWithClaims(t, "s", jwt.MapClaims{"uid": "7", "role": "admin"})
	var gotUID any
	r := newRouterWithJWT("s", func(c *gin.Context) {
		gotUID, _ = c.Get("userID")
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d want 200", w.Code)
	}
	if gotUID.(uint) != 7 {
		t.Fatalf("userID esperado 7, got %#v", gotUID)
	}
}

func TestJWT_SubFallback_OK(t *testing.T) {
	// sem uid, usa sub
	token := signWithClaims(t, "s", jwt.MapClaims{"sub": "9", "role": "admin"})
	var gotUID any
	r := newRouterWithJWT("s", func(c *gin.Context) {
		gotUID, _ = c.Get("userID")
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d want 200", w.Code)
	}
	if gotUID.(uint) != 9 {
		t.Fatalf("userID esperado 9, got %#v", gotUID)
	}
}

/* ---------- tests RequireRole ---------- */

func TestRequireRole_NoRoleForbidden(t *testing.T) {
	// token sem role
	token := signWithClaims(t, "s", jwt.MapClaims{"uid": 1})
	r := newRouterWithJWT("s", func(c *gin.Context) { c.Status(http.StatusOK) }, "admin")

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("got %d want 403", w.Code)
	}
}

func TestRequireRole_NotAllowedForbidden(t *testing.T) {
	token := signWithClaims(t, "s", jwt.MapClaims{"uid": 1, "role": "viewer"})
	r := newRouterWithJWT("s", func(c *gin.Context) { c.Status(http.StatusOK) }, "admin", "editor")

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("got %d want 403", w.Code)
	}
}

func TestRequireRole_AllowedOK(t *testing.T) {
	token := signWithClaims(t, "s", jwt.MapClaims{"uid": 1, "role": "editor"})
	r := newRouterWithJWT("s", func(c *gin.Context) { c.Status(http.StatusOK) }, "admin", "editor")

	req := httptest.NewRequest(http.MethodGet, "/api/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d want 200", w.Code)
	}
}
