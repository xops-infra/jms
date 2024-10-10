package instance_test

import (
	"testing"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/instance"
	"github.com/xops-infra/jms/model"
)

func init() {
	model.LoadYaml("/opt/jms/config.yaml")
	app.NewApp(true, "", "---").WithRobot().WithDB(false)
}

func TestServerLiveness(t *testing.T) {
	app.App.WithMcs()

	instance.LoadServer(app.App.Config)
	// instance.ServerLiveness(app.App.Config.WithSSHCheck.Alert.RobotToken)
}

// test sendMessage
func TestSendMessage(t *testing.T) {
	instance.SendMessage(app.App.Config.WithSSHCheck.Alert.RobotToken, "ssh test")
}

// instance.ServerShellRun()
func TestServerShellRun(t *testing.T) {
	instance.ServerShellRun()
}

// runShellTask
func TestRunShellTask(t *testing.T) {
	server := model.Server{
		Host: "192.168.3.233",
		Name: "test-server",
		SSHUsers: []model.SSHUser{
			{
				UserName: "root",
				Password: "111111",
			},
		},
		Port: 22,
	}
	servers := []model.Server{
		server,
		{
			Host: "192.168.16.239",
			Name: "test1-server",
			SSHUsers: []model.SSHUser{
				{
					UserName: "root",
					Password: "xxx",
				},
			},
			Port: 22,
		},
	}
	status, err := instance.RunShellTask(model.ShellTask{
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
