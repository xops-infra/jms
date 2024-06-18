package db

import (
	"errors"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/model"
)

// add broadcast
func (d *DBService) AddBroadcast(req model.CreateBroadcastRequest) error {
	if req.KeepDays == nil {
		req.KeepDays = tea.Int(9999999)
	}
	if req.Messages == nil {
		return errors.New("messages is required")
	}
	broadcast := model.Broadcast{
		Message: *req.Messages,
		Expires: time.Now().Add(time.Duration(*req.KeepDays) * 24 * time.Hour),
	}
	return d.DB.Create(&broadcast).Error
}

// get broadcast
func (d *DBService) GetBroadcast() (*model.Broadcast, error) {
	var broadcast model.Broadcast
	err := d.DB.Last(&broadcast).Error
	return &broadcast, err
}
