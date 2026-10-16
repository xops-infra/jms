package api

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/gin-gonic/gin"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newSecurityTestAPI(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	log.Default().WithFilename(filepath.Join(t.TempDir(), "api.log")).WithLevel(log.FatalLevel).Init()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := database.AutoMigrate(&model.User{}, &model.Key{}, &model.Profile{}, &model.Proxy{}, &model.Policy{}, &model.Broadcast{}); err != nil {
		t.Fatal(err)
	}
	for _, user := range []model.User{
		{ID: "member", Username: tea.String("member"), Passwd: tea.String("stored-user-password"), Groups: model.ArrayString{"engineering"}},
		{ID: "administrator", Username: tea.String("administrator"), Groups: model.ArrayString{"admin"}},
	} {
		if err := database.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
	}
	previousApp := app.App
	app.App = &app.Application{
		Config: &model.Config{WithDB: model.WithPolicy{Enable: true}, Auth: model.Auth{JWTSecret: "security-test-signing-secret"}},
		DBIo:   db.NewJmsDbService(database),
	}
	t.Cleanup(func() { app.App = previousApp })
	previousMode := gin.Mode()
	gin.SetMode(gin.TestMode)
	t.Cleanup(func() { gin.SetMode(previousMode) })
	return NewGin(), database
}

func securityTestRequest(t *testing.T, router http.Handler, method, path, username, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if username != "" {
		token, _, err := buildJWTToken(username, app.App.Config.Auth.JWTSecret, time.Hour)
		if err != nil {
			t.Fatal(err)
		}
		request.Header.Set("Authorization", bearerPrefix+token)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestAdministrativeRoutesRequireAdmin(t *testing.T) {
	router, _ := newSecurityTestAPI(t)
	routes := []struct {
		method      string
		path        string
		adminStatus int
	}{
		{http.MethodPost, "/api/v1/broadcast", http.StatusBadRequest},
		{http.MethodGet, "/api/v1/user", http.StatusOK},
		{http.MethodPost, "/api/v1/user", http.StatusBadRequest},
		{http.MethodPatch, "/api/v1/user/member", http.StatusBadRequest},
		{http.MethodPut, "/api/v1/user/member", http.StatusBadRequest},
		{http.MethodPatch, "/api/v1/approval/missing", http.StatusBadRequest},
		{http.MethodGet, "/api/v1/key", http.StatusOK},
		{http.MethodPost, "/api/v1/key", http.StatusBadRequest},
		{http.MethodDelete, "/api/v1/key/missing", http.StatusInternalServerError},
		{http.MethodGet, "/api/v1/profile", http.StatusOK},
		{http.MethodPost, "/api/v1/profile", http.StatusBadRequest},
		{http.MethodPut, "/api/v1/profile/missing", http.StatusBadRequest},
		{http.MethodDelete, "/api/v1/profile/missing", http.StatusInternalServerError},
		{http.MethodGet, "/api/v1/proxy", http.StatusOK},
		{http.MethodPost, "/api/v1/proxy", http.StatusBadRequest},
		{http.MethodPut, "/api/v1/proxy/missing", http.StatusBadRequest},
		{http.MethodDelete, "/api/v1/proxy/missing", http.StatusInternalServerError},
	}
	for _, route := range routes {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			for _, caller := range []struct {
				username string
				status   int
			}{
				{"", http.StatusUnauthorized},
				{"member", http.StatusForbidden},
				{"administrator", route.adminStatus},
			} {
				response := securityTestRequest(t, router, route.method, route.path, caller.username, "{")
				if response.Code != caller.status {
					t.Errorf("caller %q: got %d (%s), want %d", caller.username, response.Code, response.Body.String(), caller.status)
				}
			}
		})
	}
}

func TestAuthenticationRequiresActiveExactAdmin(t *testing.T) {
	router, database := newSecurityTestAPI(t)
	for _, caller := range []struct {
		name        string
		groups      model.ArrayString
		deleted     bool
		userStatus  int
		adminStatus int
	}{
		{"wildcard", model.ArrayString{"*"}, false, http.StatusOK, http.StatusForbidden},
		{"prefix", model.ArrayString{"a*"}, false, http.StatusOK, http.StatusForbidden},
		{"negated", model.ArrayString{"!other"}, false, http.StatusOK, http.StatusForbidden},
		{"literal-admin", model.ArrayString{"engineering", "admin"}, false, http.StatusOK, http.StatusOK},
		{"deleted-member", model.ArrayString{"engineering"}, true, http.StatusUnauthorized, http.StatusUnauthorized},
		{"deleted-admin", model.ArrayString{"admin"}, true, http.StatusUnauthorized, http.StatusUnauthorized},
	} {
		t.Run(caller.name, func(t *testing.T) {
			user := model.User{ID: caller.name, Username: &caller.name, Groups: caller.groups, IsDeleted: &caller.deleted}
			if err := database.Create(&user).Error; err != nil {
				t.Fatal(err)
			}
			for _, endpoint := range []struct {
				path   string
				status int
			}{
				{"/api/v1/user/me", caller.userStatus},
				{"/api/v1/policy", caller.adminStatus},
			} {
				response := securityTestRequest(t, router, http.MethodGet, endpoint.path, caller.name, "")
				if response.Code != endpoint.status {
					t.Errorf("%s: got %d (%s), want %d", endpoint.path, response.Code, response.Body.String(), endpoint.status)
				}
			}
		})
	}
	response := securityTestRequest(t, router, http.MethodGet, "/api/v1/user/me", "", "")
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous /user/me: got %d, want 401", response.Code)
	}
}

func TestApprovalUsesAuthenticatedApplicant(t *testing.T) {
	router, database := newSecurityTestAPI(t)
	body := `{"applicant":"impersonated-user","users":["beneficiary"],"server_filter":{"ip_addr":["192.0.2.10"]}}`
	response := securityTestRequest(t, router, http.MethodPost, "/api/v1/approval", "", body)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous approval: got %d, want 401", response.Code)
	}
	response = securityTestRequest(t, router, http.MethodPost, "/api/v1/approval", "member", body)
	if response.Code != http.StatusOK {
		t.Fatalf("member approval: got %d (%s), want 200", response.Code, response.Body.String())
	}
	var policy model.Policy
	if err := database.First(&policy).Error; err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(policy.Name, "member-") {
		t.Errorf("policy name %q does not use authenticated applicant", policy.Name)
	}
	if len(policy.Users) != 1 || policy.Users[0] != "beneficiary" {
		t.Errorf("approval beneficiaries changed: %v", policy.Users)
	}
	if policy.IsEnabled {
		t.Error("self-service approval must remain disabled until approved")
	}
}

func TestAPIResponsesOmitStoredCredentials(t *testing.T) {
	router, database := newSecurityTestAPI(t)
	for _, record := range []interface{}{
		&model.Key{UUID: "key-test", KeyID: "key-id", KeyName: "test.pem", Profile: "profile", PemBase64: "stored-private-key", UserName: "root"},
		&model.Profile{UUID: "profile-test", Name: "profile", AK: "stored-cloud-access-key", SK: base64.StdEncoding.EncodeToString([]byte("stored-cloud-secret")), Cloud: "aws", Regions: model.StringSlice{"test-region"}},
		&model.Proxy{UUID: "proxy-test", Name: "proxy", Host: "192.0.2.1", Port: 22, IPPrefix: "192.0.2", LoginUser: "root", LoginPasswd: "stored-proxy-password"},
	} {
		if err := database.Create(record).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, endpoint := range []struct {
		path     string
		username string
		fields   []string
	}{
		{"/api/v1/user/me", "member", []string{"passwd"}},
		{"/api/v1/user", "administrator", []string{"passwd"}},
		{"/api/v1/user?name=member", "administrator", []string{"passwd"}},
		{"/api/v1/user?group=engineering", "administrator", []string{"passwd"}},
		{"/api/v1/key", "administrator", []string{"PemBase64"}},
		{"/api/v1/profile", "administrator", []string{"AK", "SK"}},
		{"/api/v1/proxy", "administrator", []string{"login_passwd"}},
	} {
		t.Run(endpoint.path, func(t *testing.T) {
			response := securityTestRequest(t, router, http.MethodGet, endpoint.path, endpoint.username, "")
			if response.Code != http.StatusOK {
				t.Fatalf("got %d (%s), want 200", response.Code, response.Body.String())
			}
			assertNoCredentialFields(t, response.Body.Bytes(), endpoint.fields...)
		})
	}
	body := `{"name":"created-proxy","host":"192.0.2.2","port":22,"ip_prefix":"192.0.2","login_user":"root","login_passwd":"supplied-proxy-password"}`
	response := securityTestRequest(t, router, http.MethodPost, "/api/v1/proxy", "administrator", body)
	if response.Code != http.StatusOK {
		t.Fatalf("create proxy: got %d (%s), want 200", response.Code, response.Body.String())
	}
	assertNoCredentialFields(t, response.Body.Bytes(), "LoginPasswd")
	var proxy model.Proxy
	if err := database.Where("name = ?", "created-proxy").First(&proxy).Error; err != nil {
		t.Fatal(err)
	}
	if proxy.LoginPasswd != "supplied-proxy-password" {
		t.Error("request password was not preserved in storage")
	}
	internalProxy, err := app.App.DBIo.GetProxyByIP("192.0.2.10")
	if err != nil || internalProxy == nil || internalProxy.LoginPasswd == nil || *internalProxy.LoginPasswd != "stored-proxy-password" {
		t.Fatalf("internal proxy credentials not preserved: %v", err)
	}
	internalKeys, err := app.App.DBIo.InternalLoadKey()
	if err != nil || len(internalKeys) != 1 || internalKeys[0].PemBase64 == nil || *internalKeys[0].PemBase64 != "stored-private-key" {
		t.Fatalf("internal key credentials not preserved: %v", err)
	}
	internalProfiles, err := app.App.DBIo.LoadProfile()
	if err != nil || len(internalProfiles) != 1 || internalProfiles[0].SK == nil || *internalProfiles[0].SK != "stored-cloud-secret" {
		t.Fatalf("internal profile credentials not preserved: %v", err)
	}
}

func TestApprovalDecisionUsesAuthenticatedApprover(t *testing.T) {
	router, database := newSecurityTestAPI(t)
	body := `{"applicant":"member","users":["member"],"server_filter":{"ip_addr":["192.0.2.10"]}}`
	response := securityTestRequest(t, router, http.MethodPost, "/api/v1/approval", "member", body)
	if response.Code != http.StatusOK {
		t.Fatalf("create approval: got %d (%s), want 200", response.Code, response.Body.String())
	}
	var policy model.Policy
	if err := database.First(&policy).Error; err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/approval/" + policy.ID
	body = `{"applicant":"impersonated-approver","is_pass":true}`
	response = securityTestRequest(t, router, http.MethodPatch, path, "member", body)
	if response.Code != http.StatusForbidden {
		t.Fatalf("member approval decision: got %d, want 403", response.Code)
	}
	response = securityTestRequest(t, router, http.MethodPatch, path, "administrator", body)
	if response.Code != http.StatusOK {
		t.Fatalf("administrator approval decision: got %d (%s), want 200", response.Code, response.Body.String())
	}
	if err := database.First(&policy, "id = ?", policy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if !policy.IsEnabled || policy.Approver != "administrator" {
		t.Errorf("approval decision not bound to administrator: enabled=%t approver=%q", policy.IsEnabled, policy.Approver)
	}
}

func assertNoCredentialFields(t *testing.T, body []byte, fields ...string) {
	t.Helper()
	var value interface{}
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	var objects []map[string]interface{}
	switch value := value.(type) {
	case map[string]interface{}:
		objects = append(objects, value)
	case []interface{}:
		for _, item := range value {
			object, ok := item.(map[string]interface{})
			if !ok {
				t.Fatalf("unexpected response item: %T", item)
			}
			objects = append(objects, object)
		}
	default:
		t.Fatalf("unexpected response type: %T", value)
	}
	for _, object := range objects {
		for _, field := range fields {
			if _, exists := object[field]; exists {
				t.Errorf("response contains credential field %q", field)
			}
		}
	}
}
