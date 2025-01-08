package sshd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/noop/log"
)

// new audit log
func NewAuditLog(user, host string) (*os.File, error) {
	auditDir := app.App.Config.WithVideo.Dir
	logFile := fmt.Sprintf("%s/%s_%s_%s.log", auditDir, time.Now().Format("20060102_150405"), host, user)
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
	if !app.App.Config.WithVideo.Enable {
		return
	}
	days := 3 * 30 * 24 * time.Hour
	if app.App.Config.WithVideo.KeepDays > 0 {
		days = time.Duration(app.App.Config.WithVideo.KeepDays) * 24 * time.Hour
	}
	log.Debugf("days: %v", days.Hours()/24)
	// 遍历目录下的文件，删除过期文件
	filepath.Walk(app.App.Config.WithVideo.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		// 删除 scp 超过 8小时的文件 jms-tmp-file-xxx
		if strings.HasPrefix(info.Name(), "jms-tmp-file-") {
			if info.ModTime().Add(8 * time.Hour).Before(time.Now().Local()) {
				err := os.Remove(path)
				if err != nil {
					log.Errorf("Remove %s error: %s", path, err)
				}
				log.Infof("Remove more than 3 months file: %s", path)
			}
		}

		// 删除超过 3个月的文件
		if info.ModTime().Add(days).Before(time.Now().Local()) {
			err := os.Remove(path)
			if err != nil {
				log.Errorf("Remove %s error: %s", path, err)
			}
			log.Infof("Remove more than 3 months file: %s", path)
		}
		return nil
	})
}
