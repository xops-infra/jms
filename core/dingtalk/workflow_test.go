package dingtalk

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"path/filepath"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	dt "github.com/xops-infra/go-dingtalk-sdk-wrapper"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
	gossh "golang.org/x/crypto/ssh"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func newDingtalkUserTestApp(t *testing.T) *db.DBService {
	t.Helper()
	log.Default().WithFilename(filepath.Join(t.TempDir(), "dingtalk.log")).WithLevel(log.FatalLevel).Init()
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
	if err := database.AutoMigrate(&model.User{}, &model.AuthorizedKey{}); err != nil {
		t.Fatal(err)
	}
	service := db.NewJmsDbService(database)
	previousApp := app.App
	app.App = &app.Application{Config: &model.Config{WithDB: model.WithPolicy{Enable: true}}, DBIo: service}
	t.Cleanup(func() { app.App = previousApp })
	return service
}

func TestNewDingtalkUserHasNoDefaultLocalPassword(t *testing.T) {
	service := newDingtalkUserTestApp(t)
	if err := saveDingtalkUsers([]*dt.UserInfo{{Email: "directory-user@example.invalid", UserID: "directory-id", DeptIDList: []int64{42}}}); err != nil {
		t.Fatal(err)
	}
	user, err := service.DescribeActiveUser("directory-user")
	if err != nil {
		t.Fatal(err)
	}
	if user.Passwd != nil && *user.Passwd != "" {
		t.Error("directory sync created a local password")
	}
	if tea.StringValue(user.Email) != "directory-user@example.invalid" || tea.StringValue(user.DingtalkID) != "directory-id" || tea.StringValue(user.DingtalkDeptID) != "42" {
		t.Error("directory identity metadata was not provisioned")
	}
	for _, password := range []string{"directory-user@example.invalid", ""} {
		allowed, err := service.Login("directory-user", password)
		if err != nil || allowed {
			t.Errorf("unset local password authentication: allowed=%t err=%v", allowed, err)
		}
	}
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := gossh.NewPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.AddAuthorizedKey("directory-user", string(gossh.MarshalAuthorizedKey(key))); err != nil {
		t.Fatal(err)
	}
	if !service.AuthKey("directory-user", key) {
		t.Error("directory user without local password cannot use a registered SSH key")
	}
}

func TestDingtalkUserSyncPreservesExistingCredentialsAndAccountState(t *testing.T) {
	service := newDingtalkUserTestApp(t)
	for _, account := range []struct {
		name    string
		deleted bool
	}{
		{"existing", false},
		{"disabled", true},
	} {
		t.Run(account.name, func(t *testing.T) {
			password := base64.StdEncoding.EncodeToString([]byte("existing-custom-password"))
			user := model.User{
				ID: account.name, Username: &account.name, Email: tea.String(account.name + "@example.invalid"),
				Passwd: &password, Groups: model.ArrayString{"engineering"}, IsDeleted: &account.deleted,
				DingtalkID: tea.String("old-id"), DingtalkDeptID: tea.String("1"),
			}
			if err := service.DB.Create(&user).Error; err != nil {
				t.Fatal(err)
			}
			if err := saveDingtalkUsers([]*dt.UserInfo{{Email: account.name + "@example.invalid", UserID: "updated-id", DeptIDList: []int64{43}}}); err != nil {
				t.Fatal(err)
			}
			updated, err := service.DescribeUser(account.name)
			if err != nil {
				t.Fatal(err)
			}
			if updated.ID != user.ID || tea.StringValue(updated.Passwd) != password || len(updated.Groups) != 1 || updated.Groups[0] != "engineering" || tea.BoolValue(updated.IsDeleted) != account.deleted {
				t.Error("sync changed existing credentials, groups, or account state")
			}
			if tea.StringValue(updated.DingtalkID) != "updated-id" || tea.StringValue(updated.DingtalkDeptID) != "43" {
				t.Error("sync did not update directory metadata")
			}
		})
	}
}
