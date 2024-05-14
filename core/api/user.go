package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/xops-infra/jms/app"
	. "github.com/xops-infra/jms/config"
)

// @Summary 获取用户列表
// @Description 获取用户列表
// @Tags User
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param name query string false "name 支持用户名或者email查询"
// @Param group query string false "group"
// @Success 200 {object} []User
// @Router /api/v1/user [get]
func listUser(c *gin.Context) {
	name := c.Query("name")
	group := c.Query("group")
	if name != "" {
		users, err := app.App.DBService.DescribeUser(name)
		if err != nil {
			c.JSON(500, err.Error())
			return
		}
		c.JSON(200, users)
		return
	}
	if group != "" {
		users, err := app.App.DBService.QueryUserByGroup(group)
		if err != nil {
			c.JSON(500, err.Error())
			return
		}
		c.JSON(200, users)
		return
	}
	// 否则查询所有
	users, err := app.App.DBService.QueryAllUser()
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, users)
}

// @Summary 添加用户
// @Description 添加用户
// @Tags User
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param request body UserRequest true "request"
// @Success 200 {string} success
// @Router /api/v1/user [post]
func addUser(c *gin.Context) {
	var req UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	_, err := app.App.DBService.CreateUser(&req)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.String(200, "success")
}

// @Summary 追加用户组
// @Description 支持数组会与现有组进行合并
// @Tags User
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param id path string true "user id"
// @Param request body UserPatchMut true "request"
// @Success 200 {string} success
// @Router /api/v1/user/:id [patch]
func updateUserGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, fmt.Errorf("id is empty"))
		return
	}
	var req *UserPatchMut
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	if err := app.App.DBService.PatchUserGroup(id, req); err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.String(200, "success")
}

// @Summary 更新用户
// @Description 更新用户
// @Tags User
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param id path string true "user id"
// @Param request body UserRequest true "request"
// @Success 200 {string} success
// @Router /api/v1/user/:id [put]
func updateUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, fmt.Errorf("id is empty"))
		return
	}
	var req *UserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	if err := app.App.DBService.UpdateUser(id, *req); err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.String(200, "success")
}
