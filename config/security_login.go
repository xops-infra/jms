package config

import "gorm.io/gorm"

type AddSshLoginRequest struct {
	User         *string `json:"user"`          // 用户
	Client       *string `json:"client"`        // 客户端
	TargetServer *string `json:"target_server"` // 目标服务器
}

type SSHLoginRecord struct {
	gorm.Model
	User   string `gorm:"type:varchar(255);not null"` // 用户
	Client string `gorm:"type:varchar(255);not null"` // 客户端
	Target string `gorm:"type:varchar(255);not null"` // 目标服务器
}

// table name
func (SSHLoginRecord) TableName() string {
	return "record_ssh_login"
}
