package api

import (
	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
	"github.com/spf13/cast"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/model"
)

// @Summary listLoginAudit
// @Description 服务器登录审计查询，支持查询用户、IP、时间范围的日志
// @Tags audit
// @Accept json
// @Produce json
// @Param duration query int false "duration hours 24 = 1 day, 默认查 1 天的记录"
// @Param ip query string false "ip"
// @Param user query string false "user"
// @Success 200 {object} []model.SSHLoginRecord
// @Router /api/v1/audit/login [get]
func listLoginAudit(c *gin.Context) {
	req := model.QueryLoginRequest{}
	if c.Query("duration") != "" {
		req.Duration = tea.Int(cast.ToInt(c.Query("duration")))
	}
	if c.Query("ip") != "" {
		req.Ip = tea.String(c.Query("ip"))
	}
	if c.Query("user") != "" {
		req.User = tea.String(c.Query("user"))
	}

	records, err := app.App.DBIo.ListServerLoginRecord(req)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, records)
}

// @Summary listScpAudit
// @Description 服务器文件上传下载审计查询，支持上传upload,下载 download，文件名，服务器IP
// @Tags audit
// @Accept json
// @Produce json
// @Param duration query int false "duration hours 24 = 1 day, 默认查 1 天的记录"
// @Param action query string false "action"
// @Param keyword query string false "keyword"
// @Param user query string false "user"
// @Success 200 {object} []model.ScpRecord
// @Router /api/v1/audit/scp [get]
func listScpAudit(c *gin.Context) {
	req := model.QueryScpRequest{}
	if c.Query("duration") != "" {
		req.Duration = tea.Int(cast.ToInt(c.Query("duration")))
	}
	if c.Query("keyword") != "" {
		req.KeyWord = tea.String(c.Query("keyword"))
	}
	if c.Query("user") != "" {
		req.User = tea.String(c.Query("user"))
	}
	if c.Query("action") != "" {
		req.Action = tea.String(c.Query("action"))
	}
	records, err := app.App.DBIo.ListScpRecord(req)
	if err != nil {
		c.JSON(500, err.Error())
		return
	}
	c.JSON(200, records)

}
