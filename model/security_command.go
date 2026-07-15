package model

import (
	"time"

	"gorm.io/gorm"
)

// AddSSHCommandRecordRequest contains the audit fields for one non-interactive SSH command.
type AddSSHCommandRecordRequest struct {
	User           string
	KeyFingerprint string
	Client         string
	TargetServer   string
	InstanceID     string
	SSHUser        string
	SSHKey         string
	Command        string
	ExitCode       int
	Duration       time.Duration
	TimedOut       bool
}

// SSHCommandRecord records non-interactive commands executed through the bastion.
type SSHCommandRecord struct {
	gorm.Model
	UUID           string `gorm:"column:uuid;type:varchar(36);uniqueIndex;not null"`
	User           string `gorm:"column:user;type:varchar(255);not null;index"`
	KeyFingerprint string `gorm:"column:key_fingerprint;type:varchar(255);not null"`
	Client         string `gorm:"column:client;type:varchar(255);not null"`
	TargetServer   string `gorm:"column:target_server;type:varchar(255);not null;index"`
	InstanceID     string `gorm:"column:instance_id;type:varchar(255)"`
	SSHUser        string `gorm:"column:ssh_user;type:varchar(255);not null"`
	SSHKey         string `gorm:"column:ssh_key;type:varchar(255)"`
	Command        string `gorm:"column:command;type:text;not null"`
	ExitCode       int    `gorm:"column:exit_code;not null"`
	DurationMS     int64  `gorm:"column:duration_ms;not null"`
	TimedOut       bool   `gorm:"column:timed_out;not null;default:false"`
}

func (SSHCommandRecord) TableName() string {
	return "record_ssh_command"
}
