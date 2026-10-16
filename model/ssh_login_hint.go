package model

import "time"

// SSHLoginHint is learned connection state, separate from an administrator override.
type SSHLoginHint struct {
	AssetID        string `gorm:"primaryKey;size:255"`
	KeyFingerprint string `gorm:"primaryKey;size:255"`
	Address        string
	Username       string
	UpdatedAt      time.Time
}

func (SSHLoginHint) TableName() string { return "ssh_login_hint" }
