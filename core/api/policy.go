package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/model"
)

func normalizePolicyArray(values model.ArrayString) model.ArrayString {
	if len(values) == 0 {
		return model.ArrayString{}
	}
	seen := make(map[string]struct{}, len(values))
	result := make(model.ArrayString, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func normalizePolicyFilterValues(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func normalizePolicyFilter(filter *model.ServerFilterV1) bool {
	if filter == nil {
		return false
	}

	filter.ID = normalizePolicyFilterValues(filter.ID)
	filter.Name = normalizePolicyFilterValues(filter.Name)
	filter.IpAddr = normalizePolicyFilterValues(filter.IpAddr)
	filter.EnvType = normalizePolicyFilterValues(filter.EnvType)
	filter.Team = normalizePolicyFilterValues(filter.Team)

	if filter.KV != nil {
		filter.KV.Key = strings.TrimSpace(filter.KV.Key)
		filter.KV.Value = strings.TrimSpace(filter.KV.Value)
		if filter.KV.Key == "" || filter.KV.Value == "" {
			filter.KV = nil
		}
	}

	return len(filter.ID) > 0 ||
		len(filter.Name) > 0 ||
		len(filter.IpAddr) > 0 ||
		len(filter.EnvType) > 0 ||
		len(filter.Team) > 0 ||
		filter.KV != nil
}

func preparePolicyRequest(req *model.PolicyRequest) error {
	if req == nil {
		return fmt.Errorf("request is empty")
	}
	if req.Name == nil {
		return fmt.Errorf("name is empty")
	}

	name := strings.TrimSpace(*req.Name)
	if name == "" {
		return fmt.Errorf("name is empty")
	}
	req.Name = &name

	req.Users = normalizePolicyArray(req.Users)
	if len(req.Users) == 0 {
		return fmt.Errorf("users is empty")
	}

	req.Actions = normalizePolicyArray(req.Actions)
	if len(req.Actions) == 0 {
		return fmt.Errorf("actions is empty")
	}

	if !normalizePolicyFilter(req.ServerFilterV1) {
		return fmt.Errorf("server_filter is empty")
	}

	if req.ExpiresAt == nil || req.ExpiresAt.IsZero() {
		expiresAt := time.Now().Add(model.ExpireTimes[model.OneWeek])
		req.ExpiresAt = &expiresAt
	}

	if req.ApprovalID != nil {
		approvalID := strings.TrimSpace(*req.ApprovalID)
		req.ApprovalID = &approvalID
	}

	return nil
}

// @Summary 创建策略
// @Description 管理员创建策略
// @Tags Policy
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param request body PolicyRequest true "request"
// @Success 200 {string} string
// @Failure 400 {string} error
// @Failure 500 {string} error
// @Router /api/v1/policy [post]
func createPolicy(c *gin.Context) {
	var req *model.PolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	if err := preparePolicyRequest(req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	id, err := app.App.DBIo.CreatePolicy(req)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.String(200, id)
}

// @Summary 获取策略列表
// @Description 获取策略列表，只能查某人或者某个组或者某个策略，不可组合查询，查用户会带出用户所在组的策略
// @Tags Policy
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param name query string false "name"
// @Param id query string false "policy id"
// @Param user query string false "user"
// @Success 200 {object} []Policy
// @Failure 500 {string} string
// @Router /api/v1/policy [get]
func listPolicy(c *gin.Context) {
	user := c.Query("user")
	// group := c.Query("group")
	name := c.Query("name")
	id := c.Query("id")
	if user != "" {
		policies, err := app.App.DBIo.QueryPolicyByUser(user)
		if err != nil {
			c.JSON(500, err.Error())
			return
		}
		c.JSON(200, policies)
		return
	}
	if name != "" {
		policies, err := app.App.DBIo.QueryPolicyByName(name)
		if err != nil {
			c.JSON(500, err.Error())
			return
		}
		c.JSON(200, policies)
		return
	}
	if id != "" {
		policy, err := app.App.DBIo.QueryPolicyById(id)
		if err != nil {
			c.JSON(500, err.Error())
			return
		}
		c.JSON(200, policy)
		return
	}
	// 否则查询所有
	policies, err := app.App.DBIo.QueryAllPolicy()
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
// @Param request body PolicyRequest true "request"
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
	var req *model.PolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	if err := preparePolicyRequest(req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	if err := app.App.DBIo.UpdatePolicy(id, req); err != nil {
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
	if err := app.App.DBIo.DeletePolicy(id); err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.String(200, "success")
}

type PolicyCheckRequest struct {
	UserName string       `json:"username"`
	Action   model.Action `json:"action"`
	ServerIp string       `json:"serverip"`
}

// @Summary 权限校验
// @Description 权限校验，提供用户名可以查询该用户拥有的权限
// @Tags Policy
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param request body PolicyCheckRequest true "request"
// @Success 200 {object} []Policy
// @Failure 500 {string} string
// @Router /api/v1/policy/permission [post]
func checkPolicyIsOk(c *gin.Context) {
	var req *PolicyCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	c.String(200, "success")
}
