package db

import (
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
)

// 登录记录入库
func (d *DBService) AddServerLoginRecord(req *model.AddSshLoginRequest) (err error) {
	record := &model.SSHLoginRecord{
		User:   *req.User,
		Client: *req.Client,
		Target: *req.TargetServer,
	}
	return d.DB.Create(record).Error
}

// ListServerLoginRecord
func (d *DBService) ListServerLoginRecord(req model.QueryLoginRequest) (records []model.SSHLoginRecord, err error) {
	sql := d.DB.Model(&model.SSHLoginRecord{})
	log.Debugf(tea.Prettify(req))
	if req.Days != nil {
		sql = sql.Where("created_at >= ?", time.Now().AddDate(0, 0, -*req.Days))
	}
	if req.Ip != nil {
		sql = sql.Where("target = ?", *req.Ip)
	}
	if req.User != nil {
		sql = sql.Where("\"user\" = ?", *req.User)
	}
	return records, sql.Find(&records).Error
}
