package model

import "gorm.io/gorm"

// AuditSource describes the protocol entry point, not whether a caller is human.
const (
	AuditSourceWeb = "web"
	AuditSourceSSH = "ssh_interactive"
	AuditSourceCLI = "ssh_noninteractive"
)

func ValidAuditSource(source string) bool {
	return source == AuditSourceWeb || source == AuditSourceSSH || source == AuditSourceCLI || source == "unknown"
}

type QueryLoginRequest struct {
	Source   string  `json:"source"`
	User     *string `json:"user"`
	Ip       *string `json:"ip"`
	Duration *int    `json:"duration" default:"24"` // 24 hours
	Limit    *int    `json:"limit"`
	Offset   *int    `json:"offset"`
}

type AddSshLoginRequest struct {
	Source       string
	SSHUser      string
	SSHKey       string
	User         *string `json:"user"`          // 用户
	Client       *string `json:"client"`        // 客户端
	TargetServer *string `json:"target_server"` // 目标服务器
	InstanceID   *string `json:"instance_id"`   // 目标服务器实例ID
}

type SSHLoginRecord struct {
	Source  string `json:"source" gorm:"type:varchar(32);index"`
	SSHUser string `json:"ssh_user"`
	SSHKey  string `json:"ssh_key"`
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
