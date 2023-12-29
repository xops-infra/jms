package api

import (
	"github.com/gin-gonic/gin"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/policy"
)

// @Summary 获取策略列表
// @Description 获取策略列表，只能查某人或者某个组或者某个策略，不可组合查询，查用户会带出用户所在组的策略
// @Tags Policy
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param name query string false "name"
// @Param id query string false "policy id"
// @Param user query string false "user"
// @Param group query string false "group"
// @Success 200 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/policy [get]
func listPolicy(c *gin.Context) {
	user := c.Query("user")
	group := c.Query("group")
	name := c.Query("name")
	id := c.Query("id")
	if user != "" {
		policies, err := app.App.PolicyService.QueryPolicyWithGroup(user)
		if err != nil {
			c.JSON(500, NewErrorResponse(500, err.Error()))
			return
		}
		c.JSON(200, NewSuccessResponse(policies))
		return
	}
	if group != "" {
		policies, err := app.App.PolicyService.QueryPolicyByGroup(group)
		if err != nil {
			c.JSON(500, NewErrorResponse(500, err.Error()))
			return
		}
		c.JSON(200, NewSuccessResponse(policies))
		return
	}
	if name != "" {
		policies, err := app.App.PolicyService.QueryPolicyByName(name)
		if err != nil {
			c.JSON(500, Response{
				Code:    500,
				Message: err.Error(),
			})
			return
		}
		c.JSON(200, NewSuccessResponse(policies))
		return
	}
	if id != "" {
		policy, err := app.App.PolicyService.QueryPolicyById(id)
		if err != nil {
			c.JSON(500, Response{
				Code:    500,
				Message: err.Error(),
			})
			return
		}
		c.JSON(200, NewSuccessResponse(policy))
		return
	}
	// 否则查询所有
	policies, err := app.App.PolicyService.QueryAllPolicy()
	if err != nil {
		c.JSON(500, Response{
			Code:    500,
			Message: err.Error(),
		})
		return
	}
	c.JSON(200, NewSuccessResponse(policies))
}

// @Summary 更新策略
// @Description 更新策略
// @Tags Policy
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param id path string true "policy id"
// @Param request body policy.ApprovalMut true "request"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/policy/:id [put]
func updatePolicy(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, NewErrorResponse(400, "id is empty"))
		return
	}
	var req *policy.ApprovalMut
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, NewErrorResponse(400, err.Error()))
		return
	}
	if err := app.App.PolicyService.UpdatePolicy(id, req.ToPolicyMut()); err != nil {
		c.JSON(500, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(200, NewSuccessResponse(nil))
}

// @Summary 删除策略
// @Description 删除策略
// @Tags Policy
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param id path string true "policy id"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/policy/:id [delete]
func deletePolicy(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, NewErrorResponse(400, "id is empty"))
		return
	}
	if err := app.App.PolicyService.DeletePolicy(id); err != nil {
		c.JSON(500, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(200, NewSuccessResponse(nil))
}

// @Summary 创建审批策略
// @Description 创建策略
// @Tags Approval
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param request body policy.ApprovalMut true "request"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/approval [post]
func createApproval(c *gin.Context) {
	var req policy.ApprovalMut
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, NewErrorResponse(400, err.Error()))
		return
	}
	policyId, err := app.App.PolicyService.CreatePolicy(req.ToPolicyMut())
	if err != nil {
		c.JSON(500, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(200, NewSuccessResponse(gin.H{
		"policy_id": policyId,
	}))
}

// @Summary 更新审批
// @Description 更新审批结果，可以是同意或者拒绝
// @Tags Approval
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param id path string true "approval id"
// @Param request body policy.ApprovalResult true "request"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/approval/:id [patch]
func updateApproval(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, NewErrorResponse(400, "id is empty"))
		return
	}
	var req *policy.ApprovalResult
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, NewErrorResponse(400, err.Error()))
		return
	}
	if err := app.App.PolicyService.UpdatePolicyStatus(id, *req); err != nil {
		c.JSON(500, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(200, NewSuccessResponse(nil))
}

// @Summary 获取用户列表
// @Description 获取用户列表
// @Tags User
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param name query string false "name"
// @Param group query string false "group"
// @Success 200 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/user [get]
func listUser(c *gin.Context) {
	name := c.Query("name")
	group := c.Query("group")
	if name != "" {
		users, err := app.App.PolicyService.DescribeUser(name)
		if err != nil {
			c.JSON(500, NewErrorResponse(500, err.Error()))
			return
		}
		c.JSON(200, NewSuccessResponse(users))
		return
	}
	if group != "" {
		users, err := app.App.PolicyService.QueryUserByGroup(group)
		if err != nil {
			c.JSON(500, NewErrorResponse(500, err.Error()))
			return
		}
		c.JSON(200, NewSuccessResponse(users))
		return
	}
	// 否则查询所有
	users, err := app.App.PolicyService.QueryAllUser()
	if err != nil {
		c.JSON(500, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(200, NewSuccessResponse(users))
}

// @Summary 追加用户组
// @Description 支持数组会与现有组进行合并
// @Tags User
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param id path string true "user id"
// @Param request body policy.UserMut true "request"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/user/:id [patch]
func updateUserGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, NewErrorResponse(400, "id is empty"))
		return
	}
	var req *policy.UserPatchMut
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, NewErrorResponse(400, err.Error()))
		return
	}
	if err := app.App.PolicyService.PatchUserGroup(id, req); err != nil {
		c.JSON(500, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(200, NewSuccessResponse(nil))
}

// @Summary 更新用户
// @Description 更新用户
// @Tags User
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param id path string true "user id"
// @Param request body policy.UserMut true "request"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/user/:id [put]
func updateUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, NewErrorResponse(400, "id is empty"))
		return
	}
	var req *policy.UserMut
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, NewErrorResponse(400, err.Error()))
		return
	}
	if err := app.App.PolicyService.UpdateUser(id, *req); err != nil {
		c.JSON(500, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(200, NewSuccessResponse(nil))
}
