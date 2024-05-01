package db

import (
	"errors"
	"time"

	"github.com/alibabacloud-go/tea/tea"
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

// add broadcast
func (d *DBService) AddBroadcast(req CreateBroadcastRequest) error {
	if req.KeepDays == nil {
		req.KeepDays = tea.Int(9999999)
	}
	if req.Messages == nil {
		return errors.New("messages is required")
	}
	broadcast := Broadcast{
		Message: *req.Messages,
		Expires: time.Now().Add(time.Duration(*req.KeepDays) * 24 * time.Hour),
	}
	return d.DB.Create(&broadcast).Error
}

// get broadcast
func (d *DBService) GetBroadcast() (*Broadcast, error) {
	var broadcast Broadcast
	err := d.DB.Last(&broadcast).Error
	return &broadcast, err
}
