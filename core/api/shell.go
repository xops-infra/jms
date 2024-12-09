package api

import (
	"github.com/gin-gonic/gin"
	"github.com/xops-infra/jms/app"
	. "github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
)

/*
Shell API 提供了一个接口让用户能够对管理的机器执行脚本的操作。并支持查看执行的日志。
*/

// @Summary ListShellTask
// @Description list shell tasks
// @Tags shell
// @Accept json
// @Produce json
// @Success 200 {object} []ShellTask
// @Router /api/v1/shell/task [get]
func listShellTask(c *gin.Context) {
	tasks, err := app.App.JmsDBService.ListShellTask()
	if err != nil {
		c.String(500, err.Error())
		return
	}
	c.JSON(200, tasks)
}

// @Summary AddShellTask
// @Description add shell task
// @Tags shell
// @Accept json
// @Produce json
// @Param shell body CreateShellTaskRequest true "shell"
// @Success 200 {string} id
// @Router /api/v1/shell/task [post]
func addShellTask(c *gin.Context) {
	var req CreateShellTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Errorf("bind json error: %s", err)
		c.String(400, err.Error())
		return
	}
	id, err := app.App.JmsDBService.CreateShellTask(req)
	if err != nil {
		log.Errorf("create shell task error: %s", err)
		c.String(500, err.Error())
		return
	}
	c.String(200, id)
}

// @Summary UpdateShellTask
// @Description update shell task
// @Tags shell
// @Accept json
// @Produce json
// @Param shell body CreateShellTaskRequest true "shell"
// @Success 200 {string} success
// @Router /api/v1/shell/task/:uuid [put]
func updateShellTask(c *gin.Context) {
	var req CreateShellTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(400, err.Error())
		return
	}
	err := app.App.JmsDBService.UpdateShellTask(c.Param("uuid"), &req)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	c.String(200, "success")
}

// @Summary DeleteShellTask
// @Description delete shell task
// @Tags shell
// @Accept json
// @Produce json
// @Param uuid path string true "shell task uuid"
// @Success 200 {string} success
// @Router /api/v1/shell/task/:uuid [delete]
func deleteShellTask(c *gin.Context) {
	err := app.App.JmsDBService.DeleteShellTask(c.Param("uuid"))
	if err != nil {
		c.String(500, err.Error())
		return
	}
	c.String(200, "success")
}

// @Summary ListShellRecord
// @Description list shell record
// @Tags shell
// @Accept json
// @Produce json
// @Param taskid query string false "taskid"
// @Param serverip query string false "serverip"
// @Success 200 {object} []ShellTaskRecord
// @Router /api/v1/shell/record [get]
func listShellRecord(c *gin.Context) {
	taskid := c.Query("taskid")
	serverIP := c.Query("serverip")
	req := QueryRecordRequest{}
	if taskid != "" {
		req.TaskID = &taskid
	}
	if serverIP != "" {
		req.ServerIP = &serverIP
	}
	records, err := app.App.JmsDBService.QueryShellTaskRecord(&req)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	c.JSON(200, records)
}
