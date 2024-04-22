package api

import (
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
	dt "github.com/xops-infra/go-dingtalk-sdk-wrapper"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/jms/core/dingtalk"
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
// @Success 200 {object} []db.Policy
// @Failure 500 {string} string
// @Router /api/v1/policy [get]
func listPolicy(c *gin.Context) {
	user := c.Query("user")
	// group := c.Query("group")
	name := c.Query("name")
	id := c.Query("id")
	if user != "" {
		policies, err := app.App.DBService.QueryPolicyByUser(user)
		if err != nil {
			c.JSON(500, err.Error())
			return
		}
		c.JSON(200, policies)
		return
	}
	if name != "" {
		policies, err := app.App.DBService.QueryPolicyByName(name)
		if err != nil {
			c.JSON(500, err.Error())
			return
		}
		c.JSON(200, policies)
		return
	}
	if id != "" {
		policy, err := app.App.DBService.QueryPolicyById(id)
		if err != nil {
			c.JSON(500, err.Error())
			return
		}
		c.JSON(200, policy)
		return
	}
	// 否则查询所有
	policies, err := app.App.DBService.QueryAllPolicy()
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, policies)
}

// @Summary 更新策略
// @Description 更新策略
// @Tags Policy
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param id path string true "policy id"
// @Param request body db.PolicyMut true "request"
// @Success 200 {string} success
// @Failure 400 {string} error
// @Failure 500 {string} error
// @Router /api/v1/policy/:id [put]
func updatePolicy(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, fmt.Errorf("id is empty"))
		return
	}
	var req *db.PolicyMut
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	if err := app.App.DBService.UpdatePolicy(id, req); err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.String(200, "success")
}

// @Summary 删除策略
// @Description 删除策略
// @Tags Policy
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param id path string true "policy id"
// @Success 200 {string} success
// @Router /api/v1/policy/:id [delete]
func deletePolicy(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, fmt.Errorf("id is empty"))
		return
	}
	if err := app.App.DBService.DeletePolicy(id); err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.String(200, "success")
}

// @Summary 创建审批策略
// @Description 创建策略
// @Tags Approval
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param request body db.ApprovalMut true "request"
// @Success 200 {string} id
// @Router /api/v1/approval [post]
func createApproval(c *gin.Context) {
	var req db.ApprovalMut
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	// 如果启用了审批，创建审批
	if app.App.Config.WithDingtalk.Enable {
		values := []dt.FormComponentValue{}
		if req.Groups != nil {
			var vString []string
			for _, v := range req.Groups {
				vString = append(vString, v.(string))
			}
			values = append(values, dt.FormComponentValue{
				Name:  tea.String("Teams"),
				Value: tea.String(strings.Join(vString, ",")),
			})
		}
		if req.ServerFilter != nil {
			values = append(values, dt.FormComponentValue{
				Name:  tea.String("Assets"),
				Value: tea.String(tea.Prettify(req.ServerFilter)),
			})
		}
		if req.Period != nil {
			values = append(values, dt.FormComponentValue{
				Name:  tea.String("DateExpired"),
				Value: tea.String(string(*req.Period)),
			})
		}
		values = append(values, dt.FormComponentValue{
			Name:  tea.String("Comment"),
			Value: tea.String("来自API接口发起的策略申请"),
		})
		if req.Actions != nil {
			var vString []string
			for _, v := range req.Actions {
				vString = append(vString, string(v))
			}
			values = append(values, dt.FormComponentValue{
				Name:  tea.String("Actions"),
				Value: tea.String(strings.Join(vString, ",")),
			})
		}
		processid, err := dingtalk.CreateApproval(*req.Applicant, values)
		if err != nil {
			log.Errorf("dingtalk.CreateApproval error: %s", err)
			c.JSON(500, err.Error())
			return
		}
		policyId, err := app.App.DBService.CreatePolicy(req.ToPolicyMut(), &processid)
		if err != nil {
			c.JSON(500, err.Error())
			return
		}
		c.JSON(200, policyId)
	} else {
		policyId, err := app.App.DBService.CreatePolicy(req.ToPolicyMut(), nil)
		if err != nil {
			c.JSON(500, err.Error())
			return
		}
		c.String(200, policyId)
	}
}

// @Summary 更新审批
// @Description 更新审批结果，可以是同意或者拒绝
// @Tags Approval
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param id path string true "approval id"
// @Param request body db.ApprovalResult true "request"
// @Success 200 {string} success
// @Router /api/v1/approval/:id [patch]
func updateApproval(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, fmt.Errorf("id is empty"))
		return
	}
	var req *db.ApprovalResult
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	if err := app.App.DBService.UpdatePolicyStatus(id, *req); err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.String(200, "success")
}

// @Summary 获取用户列表
// @Description 获取用户列表
// @Tags User
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param name query string false "name 支持用户名或者email查询"
// @Param group query string false "group"
// @Success 200 {object} []db.User
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

// @Summary 追加用户组
// @Description 支持数组会与现有组进行合并
// @Tags User
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param id path string true "user id"
// @Param request body db.UserPatchMut true "request"
// @Success 200 {string} success
// @Router /api/v1/user/:id [patch]
func updateUserGroup(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, fmt.Errorf("id is empty"))
		return
	}
	var req *db.UserPatchMut
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
// @Param request body db.UserMut true "request"
// @Success 200 {string} success
// @Router /api/v1/user/:id [put]
func updateUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, fmt.Errorf("id is empty"))
		return
	}
	var req *db.UserRequest
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
