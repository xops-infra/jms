package db_test

import (
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/noop/log"
)

// TEST ListProxy
func TestListProxy(t *testing.T) {
	proxies, err := app.App.DBService.ListProxy()
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(proxies))
}

// TEST AddProxy
func TestAddProxy(t *testing.T) {
	proxy := db.CreateProxyRequest{
		Name:       tea.String("cas1549_prod_202"),
		Host:       tea.String("3.131.86.85"),
		Port:       tea.Int(22),
		IPPrefix:   tea.String("10.202."),
		LoginUser:  tea.String("ec2-user"),
		LoginKeyID: tea.String("cas-prod-us-east-2"),
	}
	_, err := app.App.DBService.CreateProxy(proxy)
	if err != nil {
		t.Error(err)
		return
	}
}
