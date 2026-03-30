package db

import (
	"time"

	"github.com/xops-infra/jms/model"
	"gorm.io/gorm"
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

func (d *DBService) scpAuditQuery(req model.QueryScpRequest) *gorm.DB {
	sql := d.DB.Model(&model.ScpRecord{})
	if req.Duration != nil {
		sql = sql.Where("created_at >= ?", time.Now().Add(-time.Hour*time.Duration(*req.Duration)))
	} else {
		sql = sql.Where("created_at >= ?", time.Now().AddDate(0, 0, -1))
	}
	if req.User != nil {
		sql = sql.Where("\"user\" = ?", *req.User)
	}
	if req.Action != nil && *req.Action != "" {
		sql = sql.Where("action = ?", *req.Action)
	}
	if req.KeyWord != nil && *req.KeyWord != "" {
		kw := "%" + *req.KeyWord + "%"
		sql = sql.Where("to LIKE ? OR \"from\" LIKE ?", kw, kw)
	}
	return sql
}

// ListScpRecord returns a page ordered by created_at desc and total matching rows.
func (d *DBService) ListScpRecord(req model.QueryScpRequest) (records []model.ScpRecord, total int64, err error) {
	q := d.scpAuditQuery(req)
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	limit := 50
	offset := 0
	if req.Limit != nil && *req.Limit > 0 {
		limit = *req.Limit
	}
	if req.Offset != nil && *req.Offset > 0 {
		offset = *req.Offset
	}
	err = d.scpAuditQuery(req).Order("created_at DESC").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}
