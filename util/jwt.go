package util

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
)

func GenerateJWT(id uint, payload any) (string, error) {
	ttl, _ := strconv.Atoi(os.Getenv("TOKEN_TTL"))
	claims := jwt.MapClaims{
		"id":   id,
		"iat":  time.Now().Unix(),
		"exp":  time.Now().Add(time.Second * time.Duration(ttl)).Unix(),
		"data": payload,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_PRIVATE_KEY")))
}

func ValidateJWT(c *gin.Context) error {
	t, err := getToken(c)
	if err != nil {
		return err
	}
	if !t.Valid {
		return errors.New("invalid token")
	}

	claims, ok := t.Claims.(jwt.MapClaims)
	if !ok {
		return errors.New("invalid claims")
	}

	if idFloat, ok := claims["id"].(float64); ok {
		c.Set("user_id", uint(idFloat))
	}

	if data, ok := claims["data"].(map[string]interface{}); ok {
		if role, ok := data["role"].(string); ok {
			c.Set("role", role)
		}
		if email, ok := data["email"].(string); ok {
			c.Set("email", email)
		}
	}

	return nil
}

func ExtractUserID(c *gin.Context) uint {
	id, _ := c.Get("user_id")
	if uid, ok := id.(uint); ok {
		return uid
	}
	return 0
}

func ExtractRole(c *gin.Context) string {
	return c.GetString("role")
}

func getToken(c *gin.Context) (*jwt.Token, error) {
	raw := getTokenFromRequest(c)
	if raw == "" {
		return nil, errors.New("token not provided")
	}
	return jwt.Parse(raw, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("JWT_PRIVATE_KEY")), nil
	})
}

func getTokenFromRequest(c *gin.Context) string {
	auth := c.Request.Header.Get("Authorization")
	parts := strings.Split(auth, " ")
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}
