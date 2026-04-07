package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role := c.GetString("role")
		if role == "ADMIN" {
			c.Next()
			return
		}
		for _, r := range allowedRoles {
			if r == role {
				c.Next()
				return
			}
		}
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "insufficient role"})
		c.Abort()
	}
}
