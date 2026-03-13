package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xops-infra/jms/app"
)

const (
	secret = "jms_secret"
	ttl    = 24 * time.Hour
)

// @Summary 登录
// @Description 登录接口可以换token使用。
// @Tags
// @Accept  json
// @Produce  json
// @Param user formData string true "用户名"
// @Param password formData string true "密码"
// @Router /api/v1/login [post]
func login(c *gin.Context) {

}

type LoginRequest struct {
	User     string `json:"user" form:"user" binding:"required"`
	Password string `json:"password" form:"password" binding:"required"`
}

type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// @Summary AD 登录
// @Description 使用 LDAP/AD 登录并返回 JWT token
// @Tags Auth
// @Accept  json
// @Produce  json
// @Param request body LoginRequest true "request"
// @Success 200 {object} LoginResponse
// @Failure 401 {string} string
// @Failure 500 {string} string
// @Router /api/v1/login/ad [post]
func loginAD(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBind(&req); err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	if !app.App.Config.WithLdap.Enable || app.App.Sshd.Ldap == nil {
		c.String(http.StatusServiceUnavailable, "ldap not enabled")
		return
	}
	if err := app.App.Sshd.Ldap.Login(req.User, req.Password); err != nil {
		c.String(http.StatusUnauthorized, err.Error())
		return
	}
	token, exp, err := buildJWTToken(req.User, secret, 24*time.Hour)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, LoginResponse{
		Token:     token,
		ExpiresAt: exp,
	})
}
