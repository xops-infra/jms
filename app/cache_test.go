package app_test

import (
	"github.com/xops-infra/jms/app"
)

func init() {
	app.NewApplication(true, "", "---", "/opt/jms/config.yaml").WithDB(false)
}
