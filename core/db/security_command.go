package db

import (
	"github.com/google/uuid"
	"github.com/xops-infra/jms/model"
)

// AddSSHCommandRecord persists one non-interactive SSH command audit record.
func (d *DBService) AddSSHCommandRecord(req model.AddSSHCommandRecordRequest) error {
	record := model.SSHCommandRecord{
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
