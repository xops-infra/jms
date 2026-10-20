package db

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xops-infra/jms/model"
	"gorm.io/gorm"
)

// AddSSHCommandRecord persists one non-interactive SSH command audit record.
func (d *DBService) AddSSHCommandRecord(req model.AddSSHCommandRecordRequest) error {
	record := model.SSHCommandRecord{
		Source:         model.AuditSourceCLI,
		Status:         "completed",
		UUID:           uuid.NewString(),
		User:           req.User,
		KeyFingerprint: req.KeyFingerprint,
		Client:         req.Client,
		TargetServer:   req.TargetServer,
		InstanceID:     req.InstanceID,
		SSHUser:        req.SSHUser,
		SSHKey:         req.SSHKey,
		Command:        req.Command,
		ExitCode:       req.ExitCode,
		DurationMS:     req.Duration.Milliseconds(),
		TimedOut:       req.TimedOut,
	}
	return d.DB.Create(&record).Error
}

// StartSSHCommandRecord must succeed before executing an upstream command.
func (d *DBService) StartSSHCommandRecord(req model.AddSSHCommandRecordRequest) (string, error) {
	record := model.SSHCommandRecord{
		UUID: uuid.NewString(), Source: model.AuditSourceCLI, Status: "running",
		User: req.User, KeyFingerprint: req.KeyFingerprint, Client: req.Client,
		TargetServer: req.TargetServer, InstanceID: req.InstanceID,
		SSHUser: req.SSHUser, SSHKey: req.SSHKey, Command: req.Command,
	}
	return record.UUID, d.DB.Create(&record).Error
}

// DenySSHCommandRecord records a permission decision without starting execution.
func (d *DBService) DenySSHCommandRecord(req model.AddSSHCommandRecordRequest) error {
	record := model.SSHCommandRecord{
		UUID: uuid.NewString(), Source: model.AuditSourceCLI, Status: "denied",
		User: req.User, KeyFingerprint: req.KeyFingerprint, Client: req.Client,
		TargetServer: req.TargetServer, InstanceID: req.InstanceID,
		SSHUser: req.SSHUser, SSHKey: req.SSHKey, Command: req.Command, ExitCode: 65,
	}
	return d.DB.Create(&record).Error
}

func (d *DBService) FinishSSHCommandRecord(id string, req model.AddSSHCommandRecordRequest) error {
	result := d.DB.Model(&model.SSHCommandRecord{}).Where("uuid = ? AND status = ?", id, "running").Updates(map[string]any{
		"status": "completed", "ssh_user": req.SSHUser, "ssh_key": req.SSHKey,
		"exit_code": req.ExitCode, "duration_ms": req.Duration.Milliseconds(), "timed_out": req.TimedOut,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return fmt.Errorf("command audit %s is missing or already finished", id)
	}
	return nil
}

func (d *DBService) commandAuditQuery(req model.QuerySSHCommandRequest) *gorm.DB {
	hours := req.Duration
	if hours <= 0 {
		hours = 24
	}
	q := d.DB.Model(&model.SSHCommandRecord{}).Where("created_at >= ?", time.Now().Add(-time.Duration(hours)*time.Hour))
	if req.User != "" {
		q = q.Where("\"user\" = ?", req.User)
	}
	if req.Target != "" {
		q = q.Where("target_server = ?", req.Target)
	}
	if req.Status == "failed" {
		q = q.Where("status = ? AND (exit_code <> 0 OR timed_out = ?)", "completed", true)
	} else if req.Status != "" {
		q = q.Where("status = ?", req.Status)
	}
	return q
}

func (d *DBService) ListSSHCommandRecords(req model.QuerySSHCommandRequest) (records []model.SSHCommandRecord, total int64, err error) {
	if req.Limit <= 0 {
		req.Limit = 50
	}
	if req.Limit > 200 {
		req.Limit = 200
	}
	if req.Offset < 0 {
		req.Offset = 0
	}
	if err = d.commandAuditQuery(req).Count(&total).Error; err != nil {
		return
	}
	err = d.commandAuditQuery(req).Order("created_at DESC, id DESC").Limit(req.Limit).Offset(req.Offset).Find(&records).Error
	return
}
