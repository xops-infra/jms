package api

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
)

type shellTaskResponse struct {
	UUID       string               `json:"uuid"`
	Name       string               `json:"name"`
	Shell      string               `json:"shell"`
	Corn       string               `json:"corn"`
	ExecTimes  int                  `json:"exec_times"`
	IsEnabled  bool                 `json:"is_enabled"`
	Status     model.Status         `json:"status"`
	ExecResult string               `json:"exec_result"`
	Servers    model.ServerFilterV1 `json:"servers"`
	SubmitUser string               `json:"submit_user"`
	CreatedAt  time.Time            `json:"created_at"`
	UpdatedAt  time.Time            `json:"updated_at"`
}

type shellTaskRecordResponse struct {
	UUID       string    `json:"uuid"`
	ExecTimes  int       `json:"exec_times"`
	TaskID     string    `json:"task_id"`
	TaskName   string    `json:"task_name"`
	Shell      string    `json:"shell"`
	ServerIP   string    `json:"server_ip"`
	ServerName string    `json:"server_name"`
	CostTime   string    `json:"cost_time"`
	Output     string    `json:"output"`
	IsSuccess  bool      `json:"is_success"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func buildShellTaskResponses(tasks []model.ShellTask) []shellTaskResponse {
	res := make([]shellTaskResponse, 0, len(tasks))
	for _, task := range tasks {
		res = append(res, shellTaskResponse{
			UUID:       task.UUID,
			Name:       task.Name,
			Shell:      task.Shell,
			Corn:       task.Corn,
			ExecTimes:  task.ExecTimes,
			IsEnabled:  task.IsEnabled,
			Status:     task.Status,
			ExecResult: task.ExecResult,
			Servers:    task.ServerFilter,
			SubmitUser: task.SubmitUser,
			CreatedAt:  task.CreatedAt,
			UpdatedAt:  task.UpdatedAt,
		})
	}
	return res
}

func buildShellTaskRecordResponses(records []model.ShellTaskRecord) []shellTaskRecordResponse {
	res := make([]shellTaskRecordResponse, 0, len(records))
	for _, record := range records {
		res = append(res, shellTaskRecordResponse{
			UUID:       record.UUID,
			ExecTimes:  record.ExecTimes,
			TaskID:     record.TaskID,
			TaskName:   record.TaskName,
			Shell:      record.Shell,
			ServerIP:   record.ServerIP,
			ServerName: record.ServerName,
			CostTime:   record.CostTime,
			Output:     record.Output,
			IsSuccess:  record.IsSuccess,
			CreatedAt:  record.CreatedAt,
			UpdatedAt:  record.UpdatedAt,
		})
	}
	return res
}

func writeShellTaskAdminAudit(c *gin.Context, action string, taskID, taskName *string, detail map[string]any) {
	authUser, ok := c.Get("auth_user")
	if !ok || app.App.DBIo == nil {
		return
	}
	user, ok := authUser.(model.User)
	if !ok || user.Username == nil {
		return
	}
	client := c.ClientIP()
	raw, err := json.Marshal(detail)
	if err != nil {
		log.Warnf("marshal shell task audit detail failed: %v", err)
		return
	}
	detailText := string(raw)
	if err := app.App.DBIo.AddShellTaskAuditRecord(&model.AddShellTaskAuditRequest{
		Action:   &action,
		TaskID:   taskID,
		TaskName: taskName,
		User:     user.Username,
		Client:   &client,
		Detail:   &detailText,
	}); err != nil {
		log.Warnf("create shell task admin audit failed: %v", err)
	}
}

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
	tasks, err := app.App.DBIo.ListShellTask()
	if err != nil {
		c.String(500, err.Error())
		return
	}
	writeShellTaskAdminAudit(c, "list_task", nil, nil, map[string]any{
		"count": len(tasks),
	})
	c.JSON(200, buildShellTaskResponses(tasks))
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
	var req model.CreateShellTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Errorf("bind json error: %s", err)
		c.String(400, err.Error())
		return
	}
	if authUser, ok := c.Get("auth_user"); ok {
		if user, ok := authUser.(model.User); ok && user.Username != nil {
			req.SubmitUser = user.Username
		}
	}
	id, err := app.App.DBIo.CreateShellTask(req)
	if err != nil {
		log.Errorf("create shell task error: %s", err)
		c.String(500, err.Error())
		return
	}
	writeShellTaskAdminAudit(c, "create_task", &id, req.Name, map[string]any{
		"cron":        req.Corn,
		"is_enabled":  req.IsEnabled,
		"submit_user": req.SubmitUser,
		"servers":     req.Servers,
	})
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
	var req model.CreateShellTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(400, err.Error())
		return
	}
	taskID := c.Param("uuid")
	err := app.App.DBIo.UpdateShellTask(taskID, &req)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	task, taskErr := app.App.DBIo.GetShellTask(taskID)
	if taskErr != nil {
		log.Warnf("get shell task after update failed: %v", taskErr)
	} else {
		writeShellTaskAdminAudit(c, "update_task", &task.UUID, &task.Name, map[string]any{
			"is_enabled": task.IsEnabled,
			"cron":       task.Corn,
			"status":     task.Status,
			"exec_times": task.ExecTimes,
			"servers":    task.ServerFilter,
		})
	}
	c.String(200, "success")
}

// @Summary UpdateShellTaskEnabled
// @Description enable or disable shell task
// @Tags shell
// @Accept json
// @Produce json
// @Param shell body UpdateShellTaskEnabledRequest true "shell"
// @Success 200 {string} success
// @Router /api/v1/shell/task/:uuid/enabled [patch]
func updateShellTaskEnabled(c *gin.Context) {
	var req model.UpdateShellTaskEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.String(400, err.Error())
		return
	}
	taskID := c.Param("uuid")
	err := app.App.DBIo.UpdateShellTaskEnabled(taskID, *req.IsEnabled)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	task, taskErr := app.App.DBIo.GetShellTask(taskID)
	if taskErr != nil {
		log.Warnf("get shell task after enabled update failed: %v", taskErr)
	} else {
		action := "disable_task"
		if task.IsEnabled {
			action = "enable_task"
		}
		writeShellTaskAdminAudit(c, action, &task.UUID, &task.Name, map[string]any{
			"is_enabled": task.IsEnabled,
			"status":     task.Status,
			"exec_times": task.ExecTimes,
			"cron":       task.Corn,
		})
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
	taskID := c.Param("uuid")
	task, taskErr := app.App.DBIo.GetShellTask(taskID)
	err := app.App.DBIo.DeleteShellTask(taskID)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	if taskErr != nil {
		log.Warnf("get shell task before delete failed: %v", taskErr)
	} else {
		writeShellTaskAdminAudit(c, "delete_task", &task.UUID, &task.Name, map[string]any{
			"is_enabled": task.IsEnabled,
			"cron":       task.Corn,
			"status":     task.Status,
			"exec_times": task.ExecTimes,
		})
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
	req := model.QueryRecordRequest{}
	if taskid != "" {
		req.TaskID = &taskid
	}
	if serverIP != "" {
		req.ServerIP = &serverIP
	}
	records, err := app.App.DBIo.QueryShellTaskRecord(&req)
	if err != nil {
		c.String(500, err.Error())
		return
	}
	writeShellTaskAdminAudit(c, "list_record", req.TaskID, nil, map[string]any{
		"task_id":   taskid,
		"server_ip": serverIP,
		"count":     len(records),
	})
	c.JSON(200, buildShellTaskRecordResponses(records))
}
