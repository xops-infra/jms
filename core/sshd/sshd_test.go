package sshd_test

import (
	"testing"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/sshd"
	"github.com/xops-infra/jms/model"
)

func init() {
	model.LoadYaml("/opt/jms/config.yaml")
	app.NewApp(true, "", "---")
}

func TestAuditArch(t *testing.T) {
	sshd.AuditLogArchiver()
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
	status, err := sshd.RunShellTask(model.ShellTask{
		UUID:  "xxxxxx",
		Shell: "pwd",
		Name:  "测试脚本",
		Servers: model.ServerFilterV1{
			IpAddr: []string{"*"},
		},
		Status: model.StatusPending,
	}, servers)
	if err != nil {
		t.Error(err)
	}
	t.Log(status)
}
