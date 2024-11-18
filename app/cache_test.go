package app_test

import (
	"fmt"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/app"
	model "github.com/xops-infra/jms/model"
)

func init() {
	model.LoadYaml("/opt/jms/config.yaml")
	app.NewApp(true, "", "---").WithDB(false)
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
