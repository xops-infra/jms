package db_test

import (
	"os"
	"strings"
	"testing"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/noop/log"
)

// TEST AddAuthorizedKey
// 本地认证信息迁移到数据库
func TestAddAuthorizedKey(t *testing.T) {
	// 解析本地公钥
	hostAuthorizedKeys := "/opt/jms/.ssh/authorized_keys"
	hostAuthorizedKeys = strings.Replace(hostAuthorizedKeys, "~", os.Getenv("HOME"), 1)
	data, err := os.ReadFile(hostAuthorizedKeys)
	if err != nil {
		log.Panicf(err.Error())
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if line == "" {
			continue
		}
		keyArry := strings.SplitN(line, " ", 2)
		// 入库
		err := app.App.DBService.AddAuthorizedKey(keyArry[0], keyArry[1])
		if err != nil {
			log.Panicf(err.Error())
		}
	}
}
