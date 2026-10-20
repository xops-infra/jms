package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/model"
)

func TestAuditSourcesAndCommandQueryAuthorization(t *testing.T) {
	router, database := newSecurityTestAPI(t)
	if err := database.AutoMigrate(&model.SSHLoginRecord{}, &model.SSHCommandRecord{}); err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"", model.AuditSourceWeb, model.AuditSourceSSH, model.AuditSourceCLI} {
		if err := app.App.DBIo.AddServerLoginRecord(&model.AddSshLoginRequest{
			User: tea.String("member"), Client: tea.String("192.0.2.1"), TargetServer: tea.String("192.0.2.2"), InstanceID: tea.String("asset"),
			Source: source, SSHUser: "ubuntu", SSHKey: "example.pem",
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, source := range []string{"unknown", model.AuditSourceWeb, model.AuditSourceSSH, model.AuditSourceCLI} {
		response := securityTestRequest(t, router, http.MethodGet, "/api/v1/audit/login?source="+source, "administrator", "")
		var page struct {
			Items []model.SSHLoginRecord
			Total int
		}
		if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &page) != nil || page.Total != 1 || len(page.Items) != 1 {
			t.Fatalf("source %s: %d %s", source, response.Code, response.Body)
		}
		if page.Items[0].User != "member" || page.Items[0].SSHUser != "ubuntu" {
			t.Fatal("lost actor or target identity")
		}
	}
	response := securityTestRequest(t, router, http.MethodGet, "/api/v1/audit/login?source=agent", "administrator", "")
	if response.Code != 400 {
		t.Fatalf("invalid source: %d", response.Code)
	}
	for i := 0; i < 3; i++ {
		if err := app.App.DBIo.AddSSHCommandRecord(model.AddSSHCommandRecordRequest{
			User: "member", Client: "192.0.2.1", TargetServer: "192.0.2.2", Command: "true", SSHUser: "ubuntu", ExitCode: i,
		}); err != nil {
			t.Fatal(err)
		}
	}
	for _, caller := range []struct {
		name   string
		status int
	}{{"", 401}, {"member", 403}, {"administrator", 200}} {
		response := securityTestRequest(t, router, http.MethodGet, "/api/v1/audit/command?user=member&ip=192.0.2.2&limit=1&offset=1", caller.name, "")
		if response.Code != caller.status {
			t.Fatalf("caller %q: %d %s", caller.name, response.Code, response.Body)
		}
		if caller.status == 200 {
			var page struct {
				Items                []model.SSHCommandRecord
				Total, Limit, Offset int
				HasMore              bool `json:"has_more"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &page); err != nil {
				t.Fatal(err)
			}
			if len(page.Items) != 1 || page.Total != 3 || page.Limit != 1 || page.Offset != 1 || !page.HasMore || page.Items[0].ExitCode != 1 {
				t.Fatalf("bad page: %+v", page)
			}
		}
	}
	old := model.SSHCommandRecord{UUID: "old", User: "member", SSHUser: "root", Command: "true"}
	old.CreatedAt = time.Now().Add(-48 * time.Hour)
	if err := database.Create(&old).Error; err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"?user=other", "?ip=192.0.2.99"} {
		response := securityTestRequest(t, router, http.MethodGet, "/api/v1/audit/command"+query, "administrator", "")
		var page struct{ Total int }
		if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &page) != nil || page.Total != 0 {
			t.Fatalf("filter: %s", response.Body)
		}
	}
	response = securityTestRequest(t, router, http.MethodGet, "/api/v1/audit/command?limit=999&offset=-2", "administrator", "")
	var page struct{ Total, Limit, Offset int }
	if json.Unmarshal(response.Body.Bytes(), &page) != nil || page.Total != 3 || page.Limit != 200 || page.Offset != 0 {
		t.Fatalf("bounds: %s", response.Body)
	}
	for _, value := range []string{"-1", "invalid", "9000"} {
		response = securityTestRequest(t, router, http.MethodGet, "/api/v1/audit/command?duration="+value, "administrator", "")
		if response.Code != 400 {
			t.Fatalf("duration %s: %d", value, response.Code)
		}
	}
	if err := app.App.DBIo.DenySSHCommandRecord(model.AddSSHCommandRecordRequest{User: "member", TargetServer: "192.0.2.2", Command: "true"}); err != nil {
		t.Fatal(err)
	}
	if _, err := app.App.DBIo.StartSSHCommandRecord(model.AddSSHCommandRecordRequest{User: "member", TargetServer: "192.0.2.2", Command: "true"}); err != nil {
		t.Fatal(err)
	}
	for status, want := range map[string]int{"denied": 1, "failed": 2, "running": 1, "completed": 3} {
		response = securityTestRequest(t, router, http.MethodGet, "/api/v1/audit/command?user=member&ip=192.0.2.2&status="+status, "administrator", "")
		var page struct {
			Total int
			Items []model.SSHCommandRecord
		}
		if response.Code != 200 || json.Unmarshal(response.Body.Bytes(), &page) != nil || page.Total != want {
			t.Fatalf("status %s: %d %s", status, response.Code, response.Body)
		}
		for _, row := range page.Items {
			if status != "failed" && row.Status != status {
				t.Fatalf("wrong status: %+v", row)
			}
		}
	}
	response = securityTestRequest(t, router, http.MethodGet, "/api/v1/audit/command?status=unsupported", "administrator", "")
	if response.Code != 400 {
		t.Fatalf("invalid status: %d", response.Code)
	}
}
