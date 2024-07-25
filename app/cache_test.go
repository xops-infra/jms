package app_test

import (
	"fmt"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/app"
	model "github.com/xops-infra/jms/model"
	mcsModel "github.com/xops-infra/multi-cloud-sdk/pkg/model"
)

func init() {
	model.LoadYaml("/opt/jms/config.yaml")
	app.NewSshdApplication(true, "", "---").WithDB(false)
}

// TEST QueryPolicyByUser
func TestQueryPolicyByUser(t *testing.T) {
	err := app.SetDBPolicyToCache()
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Println(tea.Prettify(app.QueryPolicyByUser("zhoushoujian")))
}

// TestPolicy 验证数据库中的策略
func TestPolicy(t *testing.T) {
	err := app.SetDBPolicyToCache()
	if err != nil {
		t.Error(err)
		return
	}

	user := model.User{
		Username: tea.String("zhoushoujian"),
		Groups:   model.ArrayString{},
	}
	server := model.Server{
		Host: "1.2.3.41",
		Name: "test-server-1",
		Tags: mcsModel.Tags{
			{
				Key:   "Test",
				Value: "zhoushoujian",
			},
		},
	}
	policies := app.QueryPolicyByUser("zhoushoujian")
	// log.Debugf("policies: %s", tea.Prettify(policies))
	fmt.Println(model.MatchPolicy(user, model.Connect, server, policies))
}
