package middleware

import (
	"net/http"
	"strings"

	"contract-scanner/internal/infra/auth"

	"github.com/gin-gonic/gin"
)

const UserIDKey = "userID"
const UsernameKey = "username"

func JWTAuth(jwtSvc auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing or invalid authorization header",
			})
			return
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := jwtSvc.Validate(token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid or expired token",
			})
			return
		}

		c.Set(UserIDKey, claims.UserID)
		c.Set(UsernameKey, claims.Username)
		c.Next()
	}
}

func GetUserID(c *gin.Context) string {
	userID, _ := c.Get(UserIDKey)
	if id, ok := userID.(string); ok {
		return id
	}
	return ""
}
