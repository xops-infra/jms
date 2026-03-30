package model

import "gorm.io/gorm"

type QueryShellTaskAuditRequest struct {
	Duration *int    `json:"duration" default:"24"`
	User     *string `json:"user"`
	Action   *string `json:"action"`
	TaskID   *string `json:"task_id"`
}

type AddShellTaskAuditRequest struct {
	Action   *string `json:"action"`
	TaskID   *string `json:"task_id"`
	TaskName *string `json:"task_name"`
	User     *string `json:"user"`
	Client   *string `json:"client"`
	Detail   *string `json:"detail"`
}

type ShellTaskAuditRecord struct {
	gorm.Model
	Action   string `json:"action" gorm:"column:action;type:varchar(255);not null"`
	TaskID   string `json:"task_id" gorm:"column:task_id;type:varchar(255)"`
	TaskName string `json:"task_name" gorm:"column:task_name;type:varchar(255)"`
	User     string `json:"user" gorm:"column:user;type:varchar(255);not null"`
	Client   string `json:"client" gorm:"column:client;type:varchar(255);not null"`
	Detail   string `json:"detail" gorm:"column:detail;type:text;not null;default:''"`
}

func (ShellTaskAuditRecord) TableName() string {
	return "record_shell_task_admin"
}
