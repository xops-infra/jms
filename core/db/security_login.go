package db

import "github.com/xops-infra/jms/model"

// 登录记录入库
func (d *DBService) AddServerLoginRecord(req *model.AddSshLoginRequest) (err error) {
	record := &model.SSHLoginRecord{
		User:   *req.User,
		Client: *req.Client,
		Target: *req.TargetServer,
	}
	return d.DB.Create(record).Error
}
