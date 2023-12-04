package utils_test

import (
	"fmt"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/utils"
	"github.com/xops-infra/noop/log"
)

func init() {
	log.Default().Init()
	app.NewApplication(true, "~/.ssh/")
}

func TestCreatePolicy(t *testing.T) {
	policy := utils.Policy{
		Id:           uuid.New().String(),
		Name:         tea.String("default"),
		IsEnabled:    tea.Bool(true),
		Users:        []string{"admin"},
		Groups:       []string{"admin"},
		ServerFilter: utils.ServerFilter{Name: tea.String("*")},
		Actions: []string{
			utils.Login.String(),
			utils.Download.String(),
			utils.Upload.String(),
		},
	}
	result := app.App.DB.Create(&policy)
	if result.Error != nil {
		t.Error(result.Error)
		return
	}
	log.Infof(tea.Prettify(policy))
}

func TestQueryPolicy(t *testing.T) {
	policies := []utils.Policy{}
	result := app.App.DB.Find(&policies)
	if result.Error != nil {
		t.Error(result.Error)
		return
	}
	for _, policy := range policies {
		fmt.Println(tea.Prettify(policy))
	}
}
