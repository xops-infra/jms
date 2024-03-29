package instance_test

import (
	"testing"

	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/instance"
)

func init() {
	log.Default().Init()
	app.NewSshdApplication(false, "~/.ssh/").WithRobot()
}

func TestServerLiveness(t *testing.T) {
	instance.LoadServer(app.App.Config)
	instance.ServerLiveness(app.App.Config.WithSSHCheck.Alert.RobotToken)
}

// test sendMessage
func TestSendMessage(t *testing.T) {
	instance.SendMessage(app.App.Config.WithSSHCheck.Alert.RobotToken, "ssh test")
}
