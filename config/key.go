package config

import "gorm.io/gorm"

type AddKeyRequest struct {
	IdentityFile *string `json:"identity_file" mapstructure:"identity_file"`              // 云上下载下来的名字，比如 jms-key.pem，private key file name
	PemBase64    *string `json:"pem_base64" binding:"required" mapstructure:"pem_base64"` // base64
	KeyID        *string `json:"key_id" binding:"required" mapstructure:"key_id"`         // 云上的key id，比如 skey-123456
	Profile      *string `json:"profile"`                                                 // 云账号的 profile，比如 aws, aliyun
}

type Key struct {
	gorm.Model `json:"-"`
	IsDelete   bool   `gorm:"column:is_delete;type:boolean;not null;default:false"`
	UUID       string `gorm:"column:uuid;type:varchar(36);unique_index;not null"`
	KeyID      string `gorm:"column:key_id;type:varchar(36);unique_index;not null"`
	KeyName    string `gorm:"column:key_name;type:varchar(255);unique_index;not null"`
	Profile    string `gorm:"column:profile;type:varchar(255);not null"`
	PemBase64  string `gorm:"column:pem_base64;type:text;not null"`
}

func (Key) TableName() string {
	return "key_table"
}
