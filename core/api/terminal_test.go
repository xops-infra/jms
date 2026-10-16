package api

import (
	"encoding/json"
	"github.com/xops-infra/jms/app"
	jmsio "github.com/xops-infra/jms/io"
	"net/http"
	"testing"

	"github.com/xops-infra/jms/model"
)

func TestSelectSSHUserPrefersRequestedKey(t *testing.T) {
	users := []model.SSHUser{
		{UserName: "root", KeyName: "default.pem"},
		{UserName: "root", KeyName: "openclaw.pem"},
	}

	selected, err := selectSSHUser(users, "root", "openclaw.pem")
	if err != nil {
		t.Fatalf("selectSSHUser returned error: %v", err)
	}
	if selected.KeyName != "openclaw.pem" {
		t.Fatalf("expected openclaw.pem, got %q", selected.KeyName)
	}
}

func TestSelectSSHUserErrorsWhenRequestedKeyMissing(t *testing.T) {
	users := []model.SSHUser{
		{UserName: "root", KeyName: "default.pem"},
	}

	_, err := selectSSHUser(users, "root", "openclaw.pem")
	if err == nil {
		t.Fatal("expected error when requested key is missing")
	}
	if got, want := err.Error(), "ssh user root with key openclaw.pem not found"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSelectSSHUserAutomaticOnlyWithoutExplicitUsername(t *testing.T) {
	users := []model.SSHUser{{UserName: "root", KeyName: "test.pem"}}
	automatic, err := selectSSHUser(users, "", "test.pem")
	if err != nil || !automatic.Auto {
		t.Fatalf("expected automatic selection: %v", err)
	}
	explicit, err := selectSSHUser(users, "root", "test.pem")
	if err != nil || explicit.Auto {
		t.Fatalf("explicit username became automatic: %v", err)
	}
}

func TestAutomaticOptionsAreOptInForCompatibleWebRollout(t *testing.T) {
	router, database := newSecurityTestAPI(t)
	if err := database.AutoMigrate(&model.Server{}); err != nil {
		t.Fatal(err)
	}
	server := model.Server{ID: "test-asset", Host: "192.0.2.10", KeyPairs: model.StringSlice{"test-key"}}
	if err := database.Create(&server).Error; err != nil {
		t.Fatal(err)
	}
	key := model.Key{UUID: "test-key", KeyID: "test-key", KeyName: "test.pem", UserName: "root", Profile: "test", PemBase64: "never-return-this-material"}
	if err := database.Create(&key).Error; err != nil {
		t.Fatal(err)
	}
	app.App.Sshd.SshdIO = jmsio.NewSshd(app.App.DBIo, nil)
	for _, tc := range []struct {
		query string
		count int
	}{{"", 1}, {"?automatic=true", 2}} {
		response := securityTestRequest(t, router, http.MethodGet, "/api/v1/servers/192.0.2.10/ssh-users"+tc.query, "administrator", "")
		if response.Code != http.StatusOK {
			t.Fatalf("unexpected status: %d %s", response.Code, response.Body)
		}
		var body struct {
			Items []sshUserItem `json:"items"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if len(body.Items) != tc.count {
			t.Fatalf("unexpected options for %q", tc.query)
		}
		if tc.query != "" && (body.Items[0].AuthType != "auto" || body.Items[0].User != "") {
			t.Fatal("automatic option missing")
		}
	}
}
