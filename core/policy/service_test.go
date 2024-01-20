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
		Users:        utils.ArrayString{tea.String("zhoushoujian")},
		Groups:       utils.ArrayString{tea.String("admin")},
		ServerFilter: &utils.ServerFilter{Name: tea.String("*")},
		Actions:      policy.All,
		ExpiresAt:    &expiredAt,
	}
	result, err := app.App.PolicyService.CreatePolicy(&req, nil)
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
		Groups: utils.ArrayString{tea.String("admin")},
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
	result, err := app.App.PolicyService.DescribeUser("zhoushoujian")
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(result))
}

func TestQueryPolicyByUser(t *testing.T) {
	result, err := app.App.PolicyService.QueryPolicyByUser("zhoushoujian")
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(result))
}
