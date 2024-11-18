package io_test

import (
	"testing"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/io"
	"github.com/xops-infra/jms/model"
)

var i *io.InstanceIO

func init() {
	model.LoadYaml("/opt/jms/config.yaml")
	app.NewApp(true, "", "---").WithRobot().WithDB(false).WithMcs()
	i = io.NewInstance(app.App.McsServer)
}

func TestServerLiveness(t *testing.T) {
	app.App.WithMcs()

	i.LoadServer(app.App.Config)
	// instance.ServerLiveness(app.App.Config.WithSSHCheck.Alert.RobotToken)
}
