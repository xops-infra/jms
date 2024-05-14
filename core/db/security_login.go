package db

import "github.com/xops-infra/jms/config"

// 登录记录入库
func (d *DBService) AddServerLoginRecord(req *config.AddSshLoginRequest) (err error) {
	record := &config.SSHLoginRecord{
		User:   *req.User,
		Client: *req.Client,
		Target: *req.TargetServer,
	}
	return d.DB.Create(record).Error
}
