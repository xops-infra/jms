package api

import (
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
	dt "github.com/xops-infra/go-dingtalk-sdk-wrapper"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/dingtalk"
	. "github.com/xops-infra/jms/model"
)

// @Summary 创建审批策略
// @Description 创建策略
// @Tags Approval
// @Accept  json
// @Produce  json
// @Param Authorization header string false "token"
// @Param request body ApprovalMut true "request"
// @Success 200 {string} id
// @Router /api/v1/approval [post]
func createApproval(c *gin.Context) {
	var req ApprovalMut
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	if req.Applicant == nil {
		c.JSON(400, fmt.Errorf("Applicant is empty"))
		return
	}
	// 如果启用了审批，创建审批
	if app.App.Config.WithDingtalk.Enable {
		values := []dt.FormComponentValue{
			{
				Name:  tea.String("EnvType"),
				Value: tea.String("prod"),
			},
		}
		if req.ServerFilter != nil {
			values = append(values, dt.FormComponentValue{
				Name:  tea.String("ServerFilter"),
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
			Value: tea.String(fmt.Sprintf("%s -来自API接口发起的策略申请", *req.Name)),
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

		policyId, err := app.App.JmsDBService.CreatePolicy(req.ToPolicyMut())
		if err != nil {
			log.Errorf("JmsDBService.CreatePolicy error: %s", err)
			c.JSON(500, err.Error())
			return
		}
		// 再创建审批
		processid, err := dingtalk.CreateApproval(*req.Applicant, values)
		if err != nil {
			log.Errorf("dingtalk.CreateApproval error: %s", err)
			// 删除策略
			if err := app.App.JmsDBService.DeletePolicy(policyId); err != nil {
				log.Errorf("JmsDBService.DeletePolicy error: %s", err)
			}
			c.JSON(500, err.Error())
			return
		}
		err = app.App.JmsDBService.UpdatePolicy(policyId, &PolicyRequest{
			ApprovalID: &processid,
		})
		if err != nil {
			log.Errorf("JmsDBService.UpdatePolicy error: %s", err)
			c.JSON(500, err.Error())
			return
		}
		c.JSON(200, policyId)
	} else {
		policyId, err := app.App.JmsDBService.CreatePolicy(req.ToPolicyMut())
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
// @Param request body ApprovalResult true "request"
// @Success 200 {string} success
// @Router /api/v1/approval/:id [patch]
func updateApproval(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(400, fmt.Errorf("id is empty"))
		return
	}
	var req *ApprovalResult
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, err.Error())
		return
	}
	if err := app.App.JmsDBService.UpdatePolicyStatus(id, *req); err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.String(200, "success")
}
