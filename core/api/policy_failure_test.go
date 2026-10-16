package api

import (
	"net/http"
	"testing"

	"github.com/xops-infra/jms/app"
	appio "github.com/xops-infra/jms/io"
	"github.com/xops-infra/jms/model"
)

func TestServerAccessFailsWhenPoliciesCannotLoad(t *testing.T) {
	router, database := newSecurityTestAPI(t)
	if err := database.AutoMigrate(&model.Server{}); err != nil {
		t.Fatal(err)
	}
	server := model.Server{ID: "target", Host: "192.0.2.10", Port: 22, KeyPairs: model.StringSlice{"managed"}}
	if err := database.Create(&server).Error; err != nil {
		t.Fatal(err)
	}
	key := model.Key{UUID: "managed", KeyID: "managed", KeyName: "managed.pem", UserName: "deploy", PemBase64: "test-key"}
	if err := database.Create(&key).Error; err != nil {
		t.Fatal(err)
	}
	app.App.Sshd.SshdIO = appio.NewSshd(app.App.DBIo, nil)
	if err := database.Migrator().DropTable(&model.Policy{}); err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{
		"/api/v1/servers",
		"/api/v1/servers/192.0.2.10/ssh-users",
		"/api/v1/terminal/ws?host=192.0.2.10&user=deploy",
	} {
		response := securityTestRequest(t, router, http.MethodGet, endpoint, "administrator", "")
		if response.Code != http.StatusServiceUnavailable {
			t.Errorf("%s: got %d (%s), want 503", endpoint, response.Code, response.Body.String())
		}
	}
}
