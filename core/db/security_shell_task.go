package db

import (
	"time"

	"github.com/xops-infra/jms/model"
)

func (d *DBService) AddShellTaskAuditRecord(req *model.AddShellTaskAuditRequest) error {
	record := &model.ShellTaskAuditRecord{
		Action: *req.Action,
		User:   *req.User,
		Client: *req.Client,
		Detail: "",
	}
	if req.TaskID != nil {
		record.TaskID = *req.TaskID
	}
	if req.TaskName != nil {
		record.TaskName = *req.TaskName
	}
	if req.Detail != nil {
		record.Detail = *req.Detail
	}
	return d.DB.Create(record).Error
}

func (d *DBService) ListShellTaskAuditRecord(req model.QueryShellTaskAuditRequest) (records []model.ShellTaskAuditRecord, err error) {
	sql := d.DB.Model(&model.ShellTaskAuditRecord{})
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
	if req.TaskID != nil {
		sql = sql.Where("task_id = ?", *req.TaskID)
	}
	return records, sql.Order("created_at desc").Find(&records).Error
}
