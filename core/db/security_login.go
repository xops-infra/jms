package db

import (
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
	"gorm.io/gorm"
)

// 登录记录入库
func (d *DBService) AddServerLoginRecord(req *model.AddSshLoginRequest) (err error) {
	record := &model.SSHLoginRecord{
		User:             *req.User,
		Client:           *req.Client,
		Target:           *req.TargetServer,
		TargetInstanceId: *req.InstanceID,
	}
	return d.DB.Create(record).Error
}

func (d *DBService) loginAuditQuery(req model.QueryLoginRequest) *gorm.DB {
	sql := d.DB.Model(&model.SSHLoginRecord{})
	if req.Duration != nil {
		sql = sql.Where("created_at >= ?", time.Now().Add(-time.Hour*time.Duration(*req.Duration)))
	} else {
		sql = sql.Where("created_at >= ?", time.Now().AddDate(0, 0, -1))
	}
	if req.Ip != nil {
		sql = sql.Where("target = ?", *req.Ip)
	}
	if req.User != nil {
		sql = sql.Where("\"user\" = ?", *req.User)
	}
	return sql
}

// ListServerLoginRecord returns a page ordered by created_at desc and total matching rows.
func (d *DBService) ListServerLoginRecord(req model.QueryLoginRequest) (records []model.SSHLoginRecord, total int64, err error) {
	log.Debugf(tea.Prettify(req))
	q := d.loginAuditQuery(req)
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
	err = d.loginAuditQuery(req).Order("created_at DESC").Offset(offset).Limit(limit).Find(&records).Error
	return records, total, err
}
