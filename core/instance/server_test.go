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
