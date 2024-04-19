package api

import "github.com/gin-gonic/gin"

// @Summary 登录
// @Description 登录
// @Tags User
// @Accept  json
// @Produce  json
// @Param user formData string true "用户名"
// @Param password formData string true "密码"
// @Router /api/v1/login [post]
func login(c *gin.Context) {

}
