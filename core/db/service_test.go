package db_test

import (
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/model"
)

func init() {
	model.LoadYaml("/opt/jms/config.yaml")
	// load env

	app.NewSshdApplication(true, "", "---").WithDB(false)
}

func TestCreatePolicy(t *testing.T) {
	expiredAt := time.Now().Add(time.Hour * 24 * 365 * 100)
	req := model.PolicyRequest{
		Name:  tea.String("zhoushoujian-test-!-manual"),
		Users: model.ArrayString{"zhoushoujian"},
		ServerFilterV1: &model.ServerFilterV1{
			IpAddr: []string{"!1.2.3.4"},
			KV: &model.KV{
				Key:   "Owner",
				Value: "zhoushoujian",
			},
		},
		Actions:   model.ConnectOnly,
		ExpiresAt: &expiredAt,
	}
	result, err := app.App.JmsDBService.CreatePolicy(&req)
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(result))
}

// TEST UpdatePolicy
func TestUpdatePolicy(t *testing.T) {
	// {"name":null,"ip_addr":["39.101.72.129"],"env_type":null,"team":null}
	req := model.PolicyRequest{
		Users: model.ArrayString{"xupeng", "fangyan",
			"fangjie", "zhangruiji", "xiayubin", "chenglinqing", "wuwentao3", "baizilong", "lushijie", "dongweijia",
			"zhonghanmeng"},
	}
	err := app.App.JmsDBService.UpdatePolicy("57284668-aad8-4104-9e4f-96c8c0186568", &req)
	if err != nil {
		t.Error(err)
		return
	}
}

func TestDeletePolicy(t *testing.T) {
	err := app.App.JmsDBService.DeletePolicy("default")
	if err != nil {
		t.Error(err)
		return
	}
}

func TestUpdateUserGroups(t *testing.T) {
	err := app.App.JmsDBService.UpdateUser("yaolong", model.UserRequest{
		Groups: model.ArrayString{"admin"},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestQueryPolicy(t *testing.T) {
	result, err := app.App.JmsDBService.QueryAllPolicy()
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(result))
}

func TestQueryUser(t *testing.T) {
	result, err := app.App.JmsDBService.DescribeUser("zhoushoujian")
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(result))
}

func TestQueryPolicyByUser(t *testing.T) {
	result, err := app.App.JmsDBService.QueryPolicyByUser("zhoushoujian")
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(result))
}

// TEST ListServerLoginRecord
func TestListServerLoginRecord(t *testing.T) {
	req := model.QueryLoginRequest{
		Duration: tea.Int(4),
		User:     tea.String("zhoushoujian"),
	}
	records, err := app.App.JmsDBService.ListServerLoginRecord(req)
	if err != nil {
		t.Error(err)
		return
	}
	for _, record := range records {

		log.Infof(tea.Prettify(record))
	}
}
