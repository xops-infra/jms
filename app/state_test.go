package app_test

import (
	"fmt"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
)

func init() {
	config.LoadYaml("/opt/jms/config.yaml")
	app.NewSshdApplication(true, "", "---")
}

// TEST SHOW CONFIG
func TestConfig(t *testing.T) {
	fmt.Println(tea.Prettify(config.Conf))
}
