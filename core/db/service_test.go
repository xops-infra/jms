package db_test

import (
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
)

func init() {
	config.LoadYaml("/opt/jms/config.yaml")
	app.NewSshdApplication(true, "", "---").WithDB()
}

func TestCreatePolicy(t *testing.T) {
	expiredAt := time.Now().Add(time.Hour * 24 * 365 * 100)
	req := config.PolicyRequest{
		Name:         tea.String("zhoushoujian-policy-1"),
		Users:        config.ArrayString{"zhoushoujian"},
		ServerFilter: &config.ServerFilter{Name: []string{"*"}},
		Actions:      config.All,
		ExpiresAt:    &expiredAt,
	}
	result, err := app.App.DBService.CreatePolicy(&req, nil)
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(result))
}

func TestDeletePolicy(t *testing.T) {
	err := app.App.DBService.DeletePolicy("default")
	if err != nil {
		t.Error(err)
		return
	}
}

func TestUpdateUserGroups(t *testing.T) {
	err := app.App.DBService.UpdateUser("yaolong", config.UserRequest{
		Groups: config.ArrayString{"admin"},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestQueryPolicy(t *testing.T) {
	result, err := app.App.DBService.QueryAllPolicy()
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(result))
}

func TestQueryUser(t *testing.T) {
	result, err := app.App.DBService.DescribeUser("zhoushoujian")
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(result))
}

func TestQueryPolicyByUser(t *testing.T) {
	result, err := app.App.DBService.QueryPolicyByUser("zhoushoujian")
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(result))
}
