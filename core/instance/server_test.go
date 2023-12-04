package instance_test

import (
	"testing"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/instance"
	"github.com/xops-infra/noop/log"
)

func init() {
	log.Default().Init()
	app.NewApplication(false, "~/.ssh/").WithDingTalk()
}

func TestServerLiveness(t *testing.T) {
	instance.LoadServer(app.App.Config)
	instance.ServerLiveness(app.App.Config.DingTalk.RobotToken)
}

// test sendMessage
func TestSendMessage(t *testing.T) {
	instance.SendMessage(app.App.Config.DingTalk.RobotToken, "ssh test")
}
