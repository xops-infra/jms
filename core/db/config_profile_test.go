package db_test

import (
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/noop/log"
)

// TEST LoadProfile
func TestLoadProfile(t *testing.T) {
	profiles, err := app.App.DBIo.LoadProfile()
	if err != nil {
		t.Error(err)
		return
	}
	log.Infof(tea.Prettify(profiles))
}
