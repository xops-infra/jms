package model

import (
	"time"

	"gorm.io/gorm"
)

type CreateBroadcastRequest struct {
	Messages *string `json:"messages" binding:"required"` // 消息内容
	KeepDays *int    `json:"keepDays"`                    // 保留天数，0 表示永久
}

type Broadcast struct {
	gorm.Model
	Message string    `gorm:"column:message;type:text;not null"`
	Expires time.Time `gorm:"column:expires;type:timestamp;not null"`
}

func (Broadcast) TableName() string {
	return "broadcast"
}
