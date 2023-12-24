package api

import (
	"github.com/gin-gonic/gin"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/policy"
)

// @Summary 获取策略列表
// @Description 获取策略列表，只能差某人或者某个组或者某个策略，不可组合查询，查用户会带出用户所在组的策略
// @Tags Policy
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param name query string false "name"
// @Param user query string false "user"
// @Param group query string false "group"
// @Success 200 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/policy [get]
func listPolicy(c *gin.Context) {
	user := c.Query("user")
	group := c.Query("group")
	name := c.Query("name")
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

// @Summary 创建策略
// @Description 创建策略
// @Tags Policy
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param request body policy.CreatePolicyRequest true "request"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /api/v1/policy [post]
func createPolicy(c *gin.Context) {
	var req policy.CreatePolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, NewErrorResponse(400, err.Error()))
		return
	}
	policyId, err := app.App.PolicyService.CreatePolicy(&req)
	if err != nil {
		c.JSON(500, NewErrorResponse(500, err.Error()))
		return
	}
	c.JSON(200, NewSuccessResponse(gin.H{
		"policy_id": policyId,
	}))
}
