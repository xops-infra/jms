package db_test

import (
	"fmt"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/app"
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
	for _, proxy := range app.App.Config.Proxys {
		fmt.Println(tea.Prettify(proxy))
		_, err := app.App.DBService.CreateProxy(proxy)
		if err != nil {
			t.Error(err)
			return
		}
	}
}
