package dingtalk

import (
	"testing"

	"github.com/xops-infra/jms/app"
)

func init() {
	app.NewApplication(true, "", "", "/opt/jms/config.yaml").WithDB(false).WithDingTalk()
}

func TestLoadDingtalkUsers(t *testing.T) {
	err := LoadUsers()
	if err != nil {
		t.Error(err)
	}
}
