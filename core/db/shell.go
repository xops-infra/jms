package db

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/robfig/cron"
	"gorm.io/gorm"
)

/*
Shell API 提供了一个接口让用户能够对管理的机器执行脚本的操作。并支持查看执行的日志。
*/

// "Pending", "Running", "Success", "Failed", "NotAllSuccess", "Cancelled"
type Status string

const (
	StatusPending       = "Pending"
	StatusRunning       = "Running"
	StatusSuccess       = "Success"
	StatusFailed        = "Failed"
	StatusNotAllSuccess = "NotAllSuccess"
	StatusCancelled     = "Cancelled"
)

type CreateShellTaskRequest struct {
	Name    *string       `json:"name" binding:"required"`    // 任务名称，唯一
	Shell   *string       `json:"shell" binding:"required"`   // 脚本内容
	Corn    *string       `json:"corn"`                       // corn表达式，支持定时执行任务，执行一次可以不传
	Servers *ServerFilter `json:"servers" binding:"required"` // 执行的机器
}

type ShellTask struct {
	gorm.Model `json:"-"`
	IsDeleted  bool         `json:"is_deleted" gorm:"column:is_deleted;default:false;not null"`
	UUID       string       `json:"uuid" gorm:"column:uuid;type:varchar(36);unique_index;not null"`
	Name       string       `json:"name" gorm:"column:name;not null,unique"`
	Shell      string       `json:"shell" gorm:"column:shell;not null"`
	Corn       string       `json:"corn" gorm:"column:cron;not null"`
	Status     Status       `json:"status" gorm:"column:status;not null"`
	Servers    ServerFilter `json:"servers" gorm:"column:servers;not null"`
	CostTime   int64        `json:"cost_time" gorm:"column:cost_time;not null"`
	SubmitUser string       `json:"submit_user" gorm:"column:submit_user;not null"` // 直接在token中获取
}

func (s *ShellTask) TableName() string {
	return "shell_task"
}

type CreateShellTaskRecordRequest struct {
	TaskID     *string `json:"task_id" binding:"required"`
	TaskName   *string `json:"task_name"`
	Shell      *string `json:"shell" binding:"required"`
	ServerIP   *string `json:"server_ip" binding:"required"`
	ServerName *string `json:"server_name"`
	CostTime   *int64  `json:"cost_time"`
	Output     *string `json:"output" binding:"required"`
}

// ShellTaskRecord 记录执行的日志
// 使用 TEXT 类型记录日志标准输出，最大支持 1G 内容足够
// 支持服务器 IP 维度，方便后续统计和查询
type ShellTaskRecord struct {
	gorm.Model `json:"-"`
	UUID       string `json:"uuid" gorm:"column:uuid;type:varchar(36);not null"`
	TaskID     string `json:"task_id" gorm:"column:task_id;not null"`
	TaskName   string `json:"task_name" gorm:"column:task_name;not null"`
	Shell      string `json:"shell" gorm:"column:shell;type:text;not null"`
	ServerIP   string `json:"server_ip" gorm:"column:server_ip;type:varchar(255);not null"`
	ServerName string `json:"server_name" gorm:"column:server_name;type:varchar(255);not null"`
	CostTime   int64  `json:"cost_time" gorm:"column:cost_time;not null"`
	Output     string `json:"output" gorm:"column:output;type:text;not null"`
}

func (s *ShellTaskRecord) TableName() string {
	return "shell_task_record"
}

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
		req.ServerIP == nil || req.Output == nil || req.CostTime == nil {
		return fmt.Errorf("invalid request. check required fields")
	}
	record := &ShellTaskRecord{
		UUID:     uuid.New().String(),
		TaskID:   *req.TaskID,
		Shell:    *req.Shell,
		ServerIP: *req.ServerIP,
		CostTime: *req.CostTime,
		Output:   *req.Output,
	}
	if req.TaskName != nil {
		record.TaskName = *req.TaskName
	}
	if req.ServerName != nil {
		record.ServerName = *req.ServerName
	}
	return d.DB.Create(&record).Error
}

type QueryRecordRequest struct {
	TaskID   *string `json:"task_id"`   // 支持依据任务 ID 查询所有记录
	ServerIP *string `json:"server_ip"` // 支持依据服务器 IP 查询所有记录
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
