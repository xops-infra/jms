package instance_test

import (
	"testing"

	"github.com/patsnapops/noop/log"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/instance"
)

func init() {
	log.Default().Init()
	app.NewApplication(false, "~/.ssh/").WithDingTalk()
}

func TestServerLiveness(t *testing.T) {
	instance.LoadServer(app.App.Config)
	instance.ServerLiveness()
}