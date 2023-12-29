package policy_test

import (
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/core/policy"
	"github.com/xops-infra/jms/utils"
)

func init() {
	log.Default().Init()
	config.Load("/opt/jms/.jms.yml")
	app.NewSshdApplication(true, "~/.ssh/").WithPolicy()
}

func TestCreatePolicy(t *testing.T) {
	expiredAt := time.Now().Add(time.Hour * 24 * 365 * 100)
	req := policy.PolicyMut{
		Name:         tea.String("zhoushoujian-policy-1"),
		Users:        utils.ArrayString{"zhoushoujian"},
		Groups:       utils.ArrayString{"admin"},
		ServerFilter: &utils.ServerFilter{Name: tea.String("*")},
		Actions: utils.ArrayString{
			policy.Connect,
			policy.Download,
			policy.Upload,
		},
		ExpiresAt: &expiredAt,
	}
	result, err := app.App.PolicyService.CreatePolicy(&req)
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(result))
}

func TestDeletePolicy(t *testing.T) {
	err := app.App.PolicyService.DeletePolicy("default")
	if err != nil {
		t.Error(err)
		return
	}
}

func TestUpdateUserGroups(t *testing.T) {
	err := app.App.PolicyService.UpdateUser("yaolong", policy.UserMut{
		Groups: utils.ArrayString{"admin"},
	})
	if err != nil {
		t.Error(err)
	}
}

func TestQueryPolicy(t *testing.T) {
	result, err := app.App.PolicyService.QueryAllPolicy()
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(result))
}

func TestQueryUser(t *testing.T) {
	result, err := app.App.PolicyService.DescribeUser("yaolong")
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(result))
}
