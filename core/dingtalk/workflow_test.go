package dingtalk

import (
	"testing"

	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
)

func init() {
	log.Default().Init()
	config.LoadYaml("/opt/jms/config.yaml")
	app.NewSshdApplication(true, "").WithPolicy().WithDingTalk()
}

func TestLoadDingtalkUsers(t *testing.T) {
	err := LoadUsers()
	if err != nil {
		t.Error(err)
	}
}
