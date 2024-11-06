package model

import "gorm.io/gorm"

type QueryScpRequest struct {
	Duration *int    `json:"duration" default:"24"` // 24 hours 默认
	KeyWord  *string `json:"keyWord"`
	User     *string `json:"user"`
	Action   *string `json:"action"`
}

type AddScpRecordRequest struct {
	Action *string `json:"action"` // download,upload
	From   *string `json:"from"`   // 来源
	To     *string `json:"to"`     // 目标
	User   *string `json:"user"`   // 用户
	Client *string `json:"client"` // 客户端
}

type ScpRecord struct {
	gorm.Model
	Action string `json:"action" gorm:"column:action;type:varchar(255);not null"` // download,upload
	From   string `json:"from" gorm:"column:from;type:varchar(255);not null"`     // 来源
	To     string `json:"to" gorm:"column:to;type:varchar(255);not null"`         // 目标
	User   string `json:"user" gorm:"column:user;type:varchar(255);not null"`     // 用户
	Client string `json:"client" gorm:"column:client;type:varchar(255);not null"` // 客户端
}

// table name
func (ScpRecord) TableName() string {
	return "record_scp"
}
