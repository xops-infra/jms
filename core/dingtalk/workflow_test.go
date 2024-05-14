package dingtalk

import (
	"testing"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
)

func init() {
	config.LoadYaml("/opt/jms/config.yaml")
	app.NewSshdApplication(true, "", "").WithDB().WithDingTalk()
}

func TestLoadDingtalkUsers(t *testing.T) {
	err := LoadUsers()
	if err != nil {
		t.Error(err)
	}
}
