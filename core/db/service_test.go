package db_test

import (
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/core/db"
)

func init() {
	log.Default().Init()
	config.LoadYaml("/opt/jms/config.yaml")
	app.NewSshdApplication(true).WithPolicy()
}

func TestCreatePolicy(t *testing.T) {
	expiredAt := time.Now().Add(time.Hour * 24 * 365 * 100)
	req := db.PolicyMut{
		Name:         tea.String("zhoushoujian-policy-1"),
		Users:        db.ArrayString{tea.String("zhoushoujian")},
		Groups:       db.ArrayString{tea.String("admin")},
		ServerFilter: &db.ServerFilter{Name: tea.String("*")},
		Actions:      db.All,
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
	err := app.App.DBService.UpdateUser("yaolong", db.UserRequest{
		Groups: db.ArrayString{tea.String("admin")},
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
