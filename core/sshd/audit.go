package sshd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/xops-infra/noop/log"

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

// audit操作日志归档定时任务
// 每天归档一次压缩到tar.gz；默认保留3个月
func AuditLogArchiver() {
	startTime := time.Now()
	defer func() {
		log.Infof("AuditLogArchiver cost: %s", time.Since(startTime))
	}()
	if !app.App.Config.APPSet.Audit.Enable {
		return
	}
	days := 3 * 30 * 24 * time.Hour
	if app.App.Config.APPSet.Audit.KeepDays > 0 {
		days = time.Duration(app.App.Config.APPSet.Audit.KeepDays) * 24 * time.Hour
	}
	log.Debugf("days: %v", days.Hours()/24)
	// 遍历目录下的文件，删除过期文件
	filepath.Walk(app.AuditDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if info.ModTime().Add(days).Before(time.Now()) {
			err := os.Remove(path)
			if err != nil {
				log.Errorf("Remove %s error: %s", path, err)
			}
			log.Infof("Remove more than 3 months file: %s", path)
		}
		return nil
	})
}
