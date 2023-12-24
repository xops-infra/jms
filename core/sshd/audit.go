package sshd

import (
	"fmt"
	"os"
	"time"

	"github.com/xops-infra/jms/app"
)

func init() {
	err := os.MkdirAll(app.AuditDir, 0755)
	if err != nil {
		panic(err)
	}
}

// new audit log
func NewAuditLog(user, host string) (*os.File, error) {
	logFile := fmt.Sprintf("%s/%s_%s_%s.log", app.AuditDir, time.Now().Format("20060102_150405"), host, user)
	logIo, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return logIo, nil
}
