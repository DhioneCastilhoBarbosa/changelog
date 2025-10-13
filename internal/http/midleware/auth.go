// internal/http/middleware/auth.go
package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Aceita "Authorization: Bearer <tok>" com case-insensitive no prefixo
		h := c.GetHeader("Authorization")
		if h == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		parts := strings.Fields(h)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		tok := parts[1]

		claims := jwt.MapClaims{}

		// Valida assinatura + algoritmo + tolerância de relógio
		token, err := jwt.ParseWithClaims(
			tok,
			claims,
			func(token *jwt.Token) (interface{}, error) { return []byte(secret), nil },
			jwt.WithLeeway(5*time.Second),
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// role (mantém comportamento atual)
		role, _ := claims["role"].(string)
		// opcional: negar sem role
		// if role == "" {
		// 	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"missing role"})
		// 	return
		// }
		c.Set("role", role)

		// uid pode vir como float64 ou string (mantém sua lógica)
		var userID uint
		switch v := claims["uid"].(type) {
		case float64:
			if v > 0 {
				userID = uint(v)
			}
		case string:
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				userID = uint(n)
			}
		}
		// fallback: aceitar "sub" como uid
		if userID == 0 {
			switch v := claims["sub"].(type) {
			case float64:
				if v > 0 {
					userID = uint(v)
				}
			case string:
				if n, err := strconv.ParseUint(v, 10, 64); err == nil {
					userID = uint(n)
				}
			}
		}
		if userID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing uid"})
			return
		}
		c.Set("userID", userID)

		// disponibiliza claims completos se precisar em handlers
		c.Set("claims", claims)

		c.Next()
	}
}

func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, r := range roles {
		allowed[r] = struct{}{}
	}
	return func(c *gin.Context) {
		roleVal, ok := c.Get("role")
		role, _ := roleVal.(string)
		if !ok || role == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if _, ok := allowed[role]; !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}
