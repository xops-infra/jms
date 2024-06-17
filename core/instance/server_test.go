package instance_test

import (
	"testing"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/core/instance"
)

func init() {
	config.LoadYaml("/opt/jms/config.yaml")
	app.NewSshdApplication(true, "", "---").WithRobot().WithDB()
}

func TestServerLiveness(t *testing.T) {
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
	server := config.Server{
		Host: "192.168.3.233",
		Name: "test-server",
		SSHUsers: []config.SSHUser{
			{
				UserName: "root",
				Password: "111111",
			},
		},
		Port: 22,
	}
	servers := []config.Server{
		server,
		{
			Host: "192.168.16.239",
			Name: "test1-server",
			SSHUsers: []config.SSHUser{
				{
					UserName: "root",
					Password: "xxx",
				},
			},
			Port: 22,
		},
	}
	status, err := instance.RunShellTask(config.ShellTask{
		UUID:  "xxxxxx",
		Shell: "pwd",
		Name:  "测试脚本",
		Servers: config.ServerFilter{
			IpAddr: []string{"*"},
		},
		Status: config.StatusPending,
	}, servers)
	if err != nil {
		t.Error(err)
	}
	t.Log(status)
}
