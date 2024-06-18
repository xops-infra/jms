package db

import "github.com/xops-infra/jms/model"

// 文件下载记录入库
func (d *DBService) AddDownloadRecord(req *model.AddScpRecordRequest) (err error) {
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
