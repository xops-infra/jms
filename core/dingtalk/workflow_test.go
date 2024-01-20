package dingtalk

import (
	"testing"

	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
)

func init() {
	log.Default().Init()
	config.Load("/opt/jms/.jms.yml")
	app.NewSshdApplication(true, "~/.ssh/").WithPolicy().WithDingTalk()
}

func TestLoadDingtalkUsers(t *testing.T) {
	err := LoadUsers()
	if err != nil {
		t.Error(err)
	}
}
