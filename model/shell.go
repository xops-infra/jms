package model

import "gorm.io/gorm"

// "Pending", "Running", "Success", "Failed", "NotAllSuccess", "Cancelled"
type Status string

const (
	StatusPending       Status = "Pending"
	StatusRunning       Status = "Running"
	StatusSuccess       Status = "Success"
	StatusFailed        Status = "Failed"
	StatusNotAllSuccess Status = "NotAllSuccess"
	StatusCancelled     Status = "Cancelled"
)

type CreateShellTaskRequest struct {
	Name    *string         `json:"name" binding:"required"`    // 任务名称，唯一
	Shell   *string         `json:"shell" binding:"required"`   // 脚本内容
	Corn    *string         `json:"corn"`                       // corn表达式，支持定时执行任务，执行一次可以不传
	Servers *ServerFilterV1 `json:"servers" binding:"required"` // 执行的机器
}

type ShellTask struct {
	gorm.Model `json:"-"`
	IsDeleted  bool           `json:"is_deleted" gorm:"column:is_deleted;default:false;not null"`
	UUID       string         `json:"uuid" gorm:"column:uuid;type:varchar(36);unique_index;not null"`
	Name       string         `json:"name" gorm:"column:name;not null,unique"`
	Shell      string         `json:"shell" gorm:"column:shell;not null"`
	Corn       string         `json:"corn" gorm:"column:cron;not null;default:''"`
	ExecTimes  int            `json:"exec_times" gorm:"column:exec_times;not null;default:0"` // 任务执行次数
	Status     Status         `json:"status" gorm:"column:status;not null"`
	ExecResult string         `json:"exec_result" gorm:"column:exec_result;type:text;not null;default:''"` // 任务执行结果信息
	Servers    ServerFilterV1 `json:"servers" gorm:"column:servers;type:json;not null"`
	CostTime   int64          `json:"cost_time" gorm:"column:cost_time;not null"`
	SubmitUser string         `json:"submit_user" gorm:"column:submit_user;not null"` // 直接在token中获取
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
	CostTime   *string `json:"cost_time"`
	ExecTimes  *int    `json:"exec_times"`                    // 任务的执行次数，取自task的执行次数字段。
	IsSuccess  *bool   `json:"is_success" binding:"required"` // 任务是否执行成功
	Output     *string `json:"output" binding:"required"`
}

// ShellTaskRecord 记录执行的日志
// 使用 TEXT 类型记录日志标准输出，最大支持 1G 内容足够
// 支持服务器 IP 维度，方便后续统计和查询
type ShellTaskRecord struct {
	gorm.Model `json:"-"`
	UUID       string `json:"uuid" gorm:"column:uuid;type:varchar(36);unique_index;not null"`
	ExecTimes  int    `json:"exec_times" gorm:"column:exec_times;not null"`
	TaskID     string `json:"task_id" gorm:"column:task_id;not null"`
	TaskName   string `json:"task_name" gorm:"column:task_name;not null"`
	Shell      string `json:"shell" gorm:"column:shell;type:text;not null"`
	ServerIP   string `json:"server_ip" gorm:"column:server_ip;type:varchar(255);not null"`
	ServerName string `json:"server_name" gorm:"column:server_name;type:varchar(255);not null"`
	CostTime   string `json:"cost_time" gorm:"column:cost_time;type:varchar(255);not null"`
	Output     string `json:"output" gorm:"column:output;type:text;not null"`
	IsSuccess  bool   `json:"is_success" gorm:"column:is_success;type:boolean;not null"`
}

func (s *ShellTaskRecord) TableName() string {
	return "shell_task_record"
}

type QueryRecordRequest struct {
	TaskID   *string `json:"task_id"`   // 支持依据任务 ID 查询所有记录
	ServerIP *string `json:"server_ip"` // 支持依据服务器 IP 查询所有记录
}
