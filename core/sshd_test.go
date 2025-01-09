package core_test

import (
	"testing"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core"
	"github.com/xops-infra/jms/model"
)

func init() {
	app.NewApplication(true, "", "---", "/opt/jms/config.yaml")
}

func TestAuditArch(t *testing.T) {
	core.AuditLogArchiver()
}

// runShellTask
func TestRunShellTask(t *testing.T) {
	server := model.Server{
		Host: "192.168.3.233",
		Name: "test-server",
		Port: 22,
	}
	servers := []model.Server{
		server,
		{
			Host: "192.168.3.234",
			Name: "test-server",
			Port: 22,
		},
	}
	keys, err := app.App.DBIo.InternalLoadKey()
	if err != nil {
		t.Error(err)
	}
	status, err := core.RunShellTask(model.ShellTask{
		UUID:  "xxxxxx",
		Shell: "pwd",
		Name:  "测试脚本",
		ServerFilter: model.ServerFilterV1{
			IpAddr: []string{"*"},
		},
		Status: model.StatusPending,
	}, servers, keys)
	if err != nil {
		t.Error(err)
	}
	t.Log(status)
}
