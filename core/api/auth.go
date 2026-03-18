package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/xops-infra/jms/app"
)

const (
	adminGroup   = "admin"
	bearerPrefix = "Bearer "
)

type jwtClaims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

// requireUser 校验 JWT 并将用户信息写入 context
func requireUser() gin.HandlerFunc {
	return func(c *gin.Context) {
		if app.App.DBIo == nil || !app.App.Config.WithDB.Enable {
			c.String(http.StatusServiceUnavailable, "db not enabled")
			c.Abort()
			return
		}
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			if token := c.Query("token"); token != "" {
				authHeader = bearerPrefix + token
			}
		}
		username, err := parseTokenUsername(authHeader, app.App.Config.Auth.JWTSecret)
		if err != nil {
			c.String(http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}
		user, err := app.App.DBIo.DescribeUser(username)
		if err != nil {
			c.String(http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}
		c.Set("auth_user", user)
		c.Next()
	}
}

func requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if app.App.DBIo == nil || !app.App.Config.WithDB.Enable {
			c.String(http.StatusServiceUnavailable, "db not enabled")
			c.Abort()
			return
		}

		username, err := parseTokenUsername(c.GetHeader("Authorization"), app.App.Config.Auth.JWTSecret)
		if err != nil {
			c.String(http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}

		user, err := app.App.DBIo.DescribeUser(username)
		if err != nil {
			c.String(http.StatusUnauthorized, "unauthorized")
			c.Abort()
			return
		}
		if user.Groups == nil || !user.Groups.Contains(adminGroup) {
			c.String(http.StatusForbidden, "admin required")
			c.Abort()
			return
		}

		c.Set("auth_user", user)
		c.Next()
	}
}

func buildJWTToken(username, secret string, ttl time.Duration) (string, int64, error) {
	now := time.Now()
	exp := now.Add(ttl).Unix()
	claims := jwtClaims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: exp,
			IssuedAt:  now.Unix(),
			Issuer:    "jms",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", 0, err
	}
	return signed, exp, nil
}

func parseTokenUsername(authHeader, secret string) (string, error) {
	if authHeader == "" || !strings.HasPrefix(authHeader, bearerPrefix) {
		return "", errors.New("missing bearer token")
	}
	raw := strings.TrimSpace(strings.TrimPrefix(authHeader, bearerPrefix))
	if raw == "" {
		return "", errors.New("empty token")
	}
	claims := &jwtClaims{}
	token, err := jwt.ParseWithClaims(raw, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		return "", errors.New("invalid token")
	}
	if claims.Username == "" {
		return "", errors.New("missing username")
	}
	return claims.Username, nil
}
