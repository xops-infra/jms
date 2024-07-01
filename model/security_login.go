package model

import "gorm.io/gorm"

type QueryLoginRequest struct {
	User     *string `json:"user"`
	Ip       *string `json:"ip"`
	Duration *int    `json:"duration" default:"24"` // 24 hours
}

type AddSshLoginRequest struct {
	User         *string `json:"user"`          // 用户
	Client       *string `json:"client"`        // 客户端
	TargetServer *string `json:"target_server"` // 目标服务器
	InstanceID   *string `json:"instance_id"`   // 目标服务器实例ID
}

type SSHLoginRecord struct {
	gorm.Model
	User             string `json:"user" gorm:"column:user;type:varchar(255);not null"`     // 用户
	Client           string `json:"client" gorm:"column:client;type:varchar(255);not null"` // 客户端
	Target           string `json:"target" gorm:"column:target;type:varchar(255);not null"` // 目标服务器
	TargetInstanceId string `json:"target_instance_id" gorm:"column:target_instance_id;type:varchar(255)"`
}

// table name
func (SSHLoginRecord) TableName() string {
	return "record_ssh_login"
}
