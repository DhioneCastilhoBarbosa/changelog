// internal/http/middleware/auth.go
package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func JWT(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
			return
		}
		tok := strings.TrimPrefix(h, "Bearer ")
		claims := jwt.MapClaims{}
		_, err := jwt.ParseWithClaims(tok, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(secret), nil
		})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
			return
		}

		// role
		role, _ := claims["role"].(string)
		if role == "" {
			// opcional: negar sem role
			// c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error":"missing role"})
			// return
		}
		c.Set("role", role)

		// uid pode vir como float64 (default do JSON) ou string
		var userID uint
		switch v := claims["uid"].(type) {
		case float64:
			if v > 0 {
				userID = uint(v)
			}
		case string:
			// tentar converter
			if n, err := strconv.ParseUint(v, 10, 64); err == nil {
				userID = uint(n)
			}
		}
		// fallback: aceitar "sub" como uid
		if userID == 0 {
			switch v := claims["sub"].(type) {
			case float64:
				userID = uint(v)
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
