package app_test

import (
	"github.com/xops-infra/jms/app"
	model "github.com/xops-infra/jms/model"
)

func init() {
	model.LoadYaml("/opt/jms/config.yaml")
	app.NewApp(true, "", "---").WithDB(false)
}
