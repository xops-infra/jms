package db

import (
	"time"

	"github.com/xops-infra/jms/model"
)

func (d *DBService) ensureShellTaskAuditSchema() error {
	d.shellTaskAuditSchemaOnce.Do(func() {
		if d.DB == nil {
			return
		}
		d.shellTaskAuditSchemaErr = d.DB.AutoMigrate(&model.ShellTaskAuditRecord{})
	})
	return d.shellTaskAuditSchemaErr
}

func (d *DBService) AddShellTaskAuditRecord(req *model.AddShellTaskAuditRequest) error {
	if err := d.ensureShellTaskAuditSchema(); err != nil {
		return err
	}
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
	if err := d.ensureShellTaskAuditSchema(); err != nil {
		return nil, err
	}
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
