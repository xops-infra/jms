package db

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/elfgzp/ssh"
	"github.com/xops-infra/jms/model"
	gossh "golang.org/x/crypto/ssh"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Run this file with the explicit source list to exclude legacy tests that initialize a live DB.
func newAuthSecurityTestService(t *testing.T) *DBService {
	t.Helper()
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
	if err := database.AutoMigrate(&model.User{}, &model.AuthorizedKey{}, &model.Policy{}); err != nil {
		t.Fatal(err)
	}
	return NewJmsDbService(database)
}

func authSecurityPublicKey(t *testing.T) ssh.PublicKey {
	t.Helper()
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := gossh.NewPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestDBAuthenticationRequiresActiveAccount(t *testing.T) {
	service := newAuthSecurityTestService(t)
	for _, account := range []struct {
		name    string
		exists  bool
		deleted bool
		allowed bool
	}{
		{"active", true, false, true},
		{"deleted", true, true, false},
		{"orphan", false, false, false},
	} {
		t.Run(account.name, func(t *testing.T) {
			if account.exists {
				user := model.User{
					ID: account.name, Username: &account.name, IsDeleted: &account.deleted,
					Passwd: tea.String(base64.StdEncoding.EncodeToString([]byte("test-password"))),
				}
				if err := service.DB.Create(&user).Error; err != nil {
					t.Fatal(err)
				}
			}
			allowed, _ := service.Login(account.name, "test-password")
			if allowed != account.allowed {
				t.Errorf("password authentication = %t, want %t", allowed, account.allowed)
			}
			key := authSecurityPublicKey(t)
			storedKey := model.AuthorizedKey{
				UUID: account.name, UserName: account.name, PublicKey: string(gossh.MarshalAuthorizedKey(key)),
			}
			if err := service.DB.Create(&storedKey).Error; err != nil {
				t.Fatal(err)
			}
			if allowed := service.AuthKey(account.name, key); allowed != account.allowed {
				t.Errorf("public key authentication = %t, want %t", allowed, account.allowed)
			}
			registration := strings.TrimSpace(string(gossh.MarshalAuthorizedKey(authSecurityPublicKey(t))))
			err := service.AddAuthorizedKey(account.name, registration)
			if (err == nil) != account.allowed {
				t.Errorf("key registration error = %v, want allowed %t", err, account.allowed)
			}
			var count int64
			if err := service.DB.Model(&model.AuthorizedKey{}).Where("public_key = ?", registration).Count(&count).Error; err != nil {
				t.Fatal(err)
			}
			if (count == 1) != account.allowed {
				t.Errorf("key registration stored %d keys, want allowed %t", count, account.allowed)
			}
		})
	}
}

func TestNeedApproveRequiresActiveLiteralAdmin(t *testing.T) {
	service := newAuthSecurityTestService(t)
	policy := model.Policy{
		ID: "pending", Name: "pending", Users: model.ArrayString{"member"},
		Actions: model.ArrayString{string(model.Connect)}, ExpiresAt: time.Now().Add(time.Hour),
	}
	if err := service.DB.Create(&policy).Error; err != nil {
		t.Fatal(err)
	}
	if err := service.DB.Model(&policy).Update("approver", nil).Error; err != nil {
		t.Fatal(err)
	}
	for _, account := range []struct {
		name    string
		groups  model.ArrayString
		deleted bool
		allowed bool
	}{
		{"wildcard", model.ArrayString{"*"}, false, false},
		{"prefix", model.ArrayString{"a*"}, false, false},
		{"negated", model.ArrayString{"!other"}, false, false},
		{"administrator", model.ArrayString{"engineering", "admin"}, false, true},
		{"deleted-administrator", model.ArrayString{"admin"}, true, false},
	} {
		t.Run(account.name, func(t *testing.T) {
			user := model.User{ID: account.name, Username: &account.name, Groups: account.groups, IsDeleted: &account.deleted}
			if err := service.DB.Create(&user).Error; err != nil {
				t.Fatal(err)
			}
			policies, err := service.NeedApprove(account.name)
			if account.deleted && err == nil {
				t.Error("deleted administrator was not rejected")
			}
			if !account.deleted && err != nil {
				t.Fatal(err)
			}
			if (len(policies) == 1) != account.allowed {
				t.Errorf("received %d pending policies, want allowed %t", len(policies), account.allowed)
			}
		})
	}
}

func TestDBLoginRejectsUnsetPassword(t *testing.T) {
	service := newAuthSecurityTestService(t)
	for _, password := range []*string{nil, tea.String("")} {
		user := model.User{ID: "no-password", Username: tea.String("no-password"), Passwd: password}
		if err := service.DB.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
		allowed, err := service.Login("no-password", "")
		if err != nil || allowed {
			t.Errorf("empty password authentication with unset credential: allowed=%t err=%v", allowed, err)
		}
		if err := service.DB.Delete(&user).Error; err != nil {
			t.Fatal(err)
		}
	}
}
