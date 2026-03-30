package sshd

import (
	"fmt"
	"os"
	"time"

	"github.com/xops-infra/jms/app"
)

// NewAuditLog creates audit log file for terminal sessions.
func NewAuditLog(user, host string) (*os.File, error) {
	auditDir := app.App.Config.WithVideo.Dir
	logFile := fmt.Sprintf("%s/%s_%s_%s.log", auditDir, time.Now().Format("20060102_150405"), host, user)
	logIo, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return logIo, nil
}
