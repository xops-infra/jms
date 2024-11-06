package db

import (
	"time"

	"github.com/xops-infra/jms/model"
)

func (d *DBService) AddScpRecord(req *model.AddScpRecordRequest) (err error) {
	record := &model.ScpRecord{
		Action: *req.Action,
		From:   *req.From,
		To:     *req.To,
		User:   *req.User,
		Client: *req.Client,
	}
	err = d.DB.Create(record).Error
	return
}

// ListScpRecord
func (d *DBService) ListScpRecord(req model.QueryScpRequest) (records []model.ScpRecord, err error) {
	sql := d.DB.Model(&model.ScpRecord{})
	if req.Duration != nil {
		sql = sql.Where("created_at >= ?", time.Now().Add(-time.Hour*time.Duration(*req.Duration)))
	} else {
		sql = sql.Where("created_at >= ?", time.Now().AddDate(0, 0, -1))
	}
	if req.User != nil {
		sql = sql.Where("\"user\" = ?", *req.User)
	}
	if req.Action != nil {
		sql = sql.Where("action = ?", *req.Action)
	}
	if req.KeyWord != nil {
		sql = sql.Where("to like ?", "%"+*req.KeyWord+"%").Where("from like ?", "%"+*req.KeyWord+"%")
	}
	return records, sql.Find(&records).Error
}
