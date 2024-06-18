package model

import "gorm.io/gorm"

type AddScpRecordRequest struct {
	Action *string `json:"action"` // download,upload
	From   *string `json:"from"`   // 来源
	To     *string `json:"to"`     // 目标
	User   *string `json:"user"`   // 用户
	Client *string `json:"client"` // 客户端
}

type ScpRecord struct {
	gorm.Model
	Action string `gorm:"type:varchar(255);not null"` // download,upload
	From   string `gorm:"type:varchar(255);not null"` // 来源
	To     string `gorm:"type:varchar(255);not null"` // 目标
	User   string `gorm:"type:varchar(255);not null"` // 用户
	Client string `gorm:"type:varchar(255);not null"` // 客户端
}

// table name
func (ScpRecord) TableName() string {
	return "record_scp"
}
