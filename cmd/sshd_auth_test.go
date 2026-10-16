package cmd

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/elfgzp/ssh"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
	gossh "golang.org/x/crypto/ssh"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type sshAuthTestContext struct {
	ssh.Context
	username string
}

func (c sshAuthTestContext) User() string { return c.username }

func newSSHAuthTestApp(t *testing.T) *gorm.DB {
	t.Helper()
	log.Default().WithFilename(filepath.Join(t.TempDir(), "sshd.log")).WithLevel(log.FatalLevel).Init()
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
	for _, user := range []model.User{
		{ID: "active", Username: tea.String("active"), Passwd: tea.String(base64.StdEncoding.EncodeToString([]byte("test-password")))},
		{ID: "deleted", Username: tea.String("deleted"), IsDeleted: tea.Bool(true), Passwd: tea.String(base64.StdEncoding.EncodeToString([]byte("test-password")))},
	} {
		if err := database.Create(&user).Error; err != nil {
			t.Fatal(err)
		}
	}
	previousApp := app.App
	app.App = &app.Application{
		Config: &model.Config{WithDB: model.WithPolicy{Enable: true}},
		DBIo:   db.NewJmsDbService(database), SSHDir: t.TempDir() + string(os.PathSeparator),
	}
	t.Cleanup(func() { app.App = previousApp })
	return database
}

func TestSSHPasswordAuthenticationRequiresActiveAccount(t *testing.T) {
	newSSHAuthTestApp(t)
	for _, account := range []struct {
		name     string
		password string
		allowed  bool
	}{
		{"active", "test-password", true},
		{"active", "wrong-password", false},
		{"deleted", "test-password", false},
		{"missing", "test-password", false},
	} {
		if allowed := passwordAuth(sshAuthTestContext{username: account.name}, account.password); allowed != account.allowed {
			t.Errorf("%s password authentication = %t, want %t", account.name, allowed, account.allowed)
		}
	}
	app.App.Config.WithDB.Enable = false
	if !passwordAuth(sshAuthTestContext{username: "jms"}, "jms") {
		t.Error("noDB default authentication changed")
	}
	if passwordAuth(sshAuthTestContext{username: "active"}, "test-password") {
		t.Error("noDB authentication unexpectedly used DB credentials")
	}
}

func TestLDAPSSHAuthenticationChecksAccountBeforeLDAP(t *testing.T) {
	newSSHAuthTestApp(t)
	app.App.Config.WithLdap.Enable = true
	user := model.User{ID: "ldap-without-local-password", Username: tea.String("ldap-without-local-password")}
	if err := app.App.DBIo.DB.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	for _, account := range []struct {
		name    string
		allowed bool
	}{
		{"active", true},
		{"ldap-without-local-password", true},
		{"deleted", false},
		{"missing", false},
	} {
		called := false
		ldapLogin := func(username, password string) error {
			called = true
			if username != account.name || password != "ldap-password" {
				t.Error("LDAP received changed credentials")
			}
			return nil
		}
		if allowed := authenticateSSHPassword(account.name, "ldap-password", ldapLogin); allowed != account.allowed {
			t.Errorf("%s LDAP authentication = %t, want %t", account.name, allowed, account.allowed)
		}
		if called != account.allowed {
			t.Errorf("%s LDAP invoked = %t, want %t", account.name, called, account.allowed)
		}
	}
	app.App.Config.WithDB.Enable = false
	called := false
	if !authenticateSSHPassword("ldap-only-user", "ldap-password", func(string, string) error { called = true; return nil }) || !called {
		t.Error("noDB LDAP authentication changed")
	}
}

func TestSSHPublicKeyAuthenticationRequiresActiveAccount(t *testing.T) {
	database := newSSHAuthTestApp(t)
	public, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	key, err := gossh.NewPublicKey(public)
	if err != nil {
		t.Fatal(err)
	}
	for _, account := range []struct {
		name    string
		allowed bool
	}{
		{"active", true},
		{"deleted", false},
		{"orphan", false},
	} {
		record := model.AuthorizedKey{UUID: account.name, UserName: account.name, PublicKey: string(gossh.MarshalAuthorizedKey(key))}
		if err := database.Create(&record).Error; err != nil {
			t.Fatal(err)
		}
		if allowed := publicKeyAuth(sshAuthTestContext{username: account.name}, key); allowed != account.allowed {
			t.Errorf("%s public key authentication = %t, want %t", account.name, allowed, account.allowed)
		}
	}
	app.App.Config.WithDB.Enable = false
	if err := os.WriteFile(filepath.Join(app.App.SSHDir, "authorized_keys"), append([]byte("file-user "), gossh.MarshalAuthorizedKey(key)...), 0600); err != nil {
		t.Fatal(err)
	}
	if !publicKeyAuth(sshAuthTestContext{username: "file-user"}, key) {
		t.Error("noDB file-based public key authentication changed")
	}
}
