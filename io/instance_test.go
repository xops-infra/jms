package io_test

import (
	"testing"

	"github.com/xops-infra/jms/app"
)

func init() {
	app.NewApplication(true, "", "---", "/opt/jms/config.yaml").WithRobot().WithDB(true).WithMcs()
}

func TestServerLiveness(t *testing.T) {
	app.App.Scheduler.InstanceIO.LoadServer()
}
