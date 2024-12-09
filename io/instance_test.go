package io_test

import (
	"testing"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/model"
)

func init() {
	model.LoadYaml("/opt/jms/config.yaml")
	app.NewApp(true, "", "---").WithRobot().WithDB(true).WithMcs()
}

func TestServerLiveness(t *testing.T) {
	app.App.Core.InstanceIO.LoadServer()
}
