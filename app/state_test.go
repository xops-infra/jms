package app_test

import (
	"fmt"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/noop/log"
)

func init() {
	log.Default().Init()
	config.LoadYaml("/opt/jms/.jms.yml")
	app.NewSshdApplication(true, "~/.ssh/")
}

// TEST SHOW CONFIG
func TestConfig(t *testing.T) {
	fmt.Println(tea.Prettify(config.Conf))
}
