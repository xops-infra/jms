package db_test

import (
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/noop/log"
)

// TEST ListProxy
func TestListProxy(t *testing.T) {
	proxies, err := app.App.JmsDBService.ListProxy()
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(proxies))
}
