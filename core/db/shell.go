package db

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/robfig/cron"
	. "github.com/xops-infra/jms/model"
)

/*
Shell API 提供了一个接口让用户能够对管理的机器执行脚本的操作。并支持查看执行的日志。
*/

func (d *DBService) CreateShellTask(req CreateShellTaskRequest) (string, error) {
	if req.Name == nil || req.Shell == nil || req.Servers == nil {
		return "", fmt.Errorf("invalid request. check required fields")
	}
	// 校验 Corn
	if req.Corn != nil {
		if _, err := cron.Parse(*req.Corn); err != nil {
			return "", fmt.Errorf("invalid corn expression: %v", err)
		}
	}
	// 先查询是否存在
	var count int64
	err := d.DB.Model(ShellTask{}).Where("name = ? and is_deleted is false", *req.Name).Count(&count).Error
	if err != nil {
		return "", fmt.Errorf("failed to query shell task: %v", err)
	}
	if count > 0 {
		return "", fmt.Errorf("shell task name %s already exists", *req.Name)
	}
	task := &ShellTask{
		UUID:    uuid.New().String(),
		Name:    *req.Name,
		Shell:   *req.Shell,
		Servers: *req.Servers,
		Status:  StatusPending,
	}
	if req.Corn != nil {
		task.Corn = *req.Corn
	}
	err = d.DB.Create(&task).Error
	return task.UUID, err
}

// TODO: 支持条件查询
func (d *DBService) ListShellTask() ([]ShellTask, error) {
	var tasks []ShellTask
	err := d.DB.Where("is_deleted is false").Find(&tasks).Order("created_at").Error
	return tasks, err
}

func (d *DBService) GetShellTask(uuid string) (*ShellTask, error) {
	var task ShellTask
	err := d.DB.Where("uuid = ? and is_deleted is false", uuid).First(&task).Error
	return &task, err
}

func (d *DBService) UpdateShellTask(uuid string, req *CreateShellTaskRequest) error {
	var task ShellTask
	err := d.DB.Where("uuid = ? and is_deleted is false", uuid).First(&task).Error
	if err != nil {
		return err
	}
	// 更新
	if req.Name != nil {
		task.Name = *req.Name
	}
	if req.Shell != nil {
		task.Shell = *req.Shell
	}
	if req.Servers != nil {
		task.Servers = *req.Servers
	}
	if req.Corn != nil {
		task.Corn = *req.Corn
	}
	err = d.DB.Save(&task).Error
	return err
}

func (d *DBService) UpdateShellTaskStatus(uuid string, status Status, output string) error {
	var task ShellTask
	err := d.DB.Where("uuid = ? and is_deleted is false", uuid).First(&task).Error
	if err != nil {
		return err
	}
	task.Status = status
	task.ExecResult = output
	if status == StatusRunning {
		task.ExecTimes++
	}
	return d.DB.Save(&task).Error
}

func (d *DBService) DeleteShellTask(uuid string) error {
	// 先查询是否存在
	var task ShellTask
	err := d.DB.Where("uuid = ? and is_deleted is false", uuid).First(&task).Error
	if err != nil {
		return err
	}
	task.IsDeleted = true
	return d.DB.Save(&task).Error
}

func (d *DBService) CreateShellTaskRecord(req *CreateShellTaskRecordRequest) error {
	if req.TaskID == nil || req.Shell == nil ||
		req.ServerIP == nil || req.Output == nil || req.CostTime == nil || req.IsSuccess == nil {
		return fmt.Errorf("invalid request. check required fields")
	}
	record := &ShellTaskRecord{
		UUID:      uuid.New().String(),
		TaskID:    *req.TaskID,
		Shell:     *req.Shell,
		ServerIP:  *req.ServerIP,
		CostTime:  *req.CostTime,
		Output:    *req.Output,
		IsSuccess: *req.IsSuccess,
		ExecTimes: *req.ExecTimes,
	}
	if req.TaskName != nil {
		record.TaskName = *req.TaskName
	}
	if req.ServerName != nil {
		record.ServerName = *req.ServerName
	}
	return d.DB.Create(&record).Error
}

func (d *DBService) QueryShellTaskRecord(query *QueryRecordRequest) ([]ShellTaskRecord, error) {
	var records []ShellTaskRecord
	sql := d.DB.Model(&ShellTaskRecord{})
	if query.TaskID != nil {
		sql = sql.Where("task_id = ?", *query.TaskID)
	}
	if query.ServerIP != nil {
		sql = sql.Where("server_ip = ?", *query.ServerIP)
	}
	err := sql.Order("created_at").Find(&records).Error
	return records, err
}
