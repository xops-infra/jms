package io

import (
	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path/filepath"
	"testing"
)

func TestServerUsernameOverridesManagedKeyOnlyForThatAsset(t *testing.T) {
	log.Default().WithFilename(filepath.Join(t.TempDir(), "ssh.log")).WithLevel(log.FatalLevel).Init()
	connection, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "test.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.AutoMigrate(&model.Server{}, &model.Key{}); err != nil {
		t.Fatal(err)
	}
	for _, server := range []model.Server{{ID: "overridden", Host: "192.0.2.1", User: " ubuntu ", KeyPairs: model.StringSlice{"shared-key"}, Profile: "test"}, {ID: "unchanged", Host: "192.0.2.2", KeyPairs: model.StringSlice{"shared-key"}, Profile: "test"}, {ID: "fallback", Host: "192.0.2.3", User: "ubuntu", Profile: "test"}} {
		if err := connection.Create(&server).Error; err != nil {
			t.Fatal(err)
		}
	}
	key := model.Key{UUID: "test-key", KeyID: "shared-key", KeyName: "test.pem", UserName: "root", Profile: "test", PemBase64: "test-managed-material"}
	if err := connection.Create(&key).Error; err != nil {
		t.Fatal(err)
	}
	service := NewSshd(db.NewJmsDbService(connection), nil)
	for _, tc := range []struct{ host, user string }{{"192.0.2.1", "ubuntu"}, {"192.0.2.2", "root"}, {"192.0.2.3", "ubuntu"}} {
		_, users, err := service.GetSSHUsersByHostLive(tc.host)
		if err != nil {
			t.Fatal(err)
		}
		if len(users) != 1 || users[0].UserName != tc.user || users[0].KeyName != "test.pem" || users[0].Base64Pem != "test-managed-material" {
			t.Fatalf("unexpected resolved identity for %s: count=%d", tc.host, len(users))
		}
	}
	for _, selector := range []string{"ubuntu@192.0.2.1:/tmp/test", "ubuntu@192.0.2.1#key_name=test.pem:/tmp/test"} {
		user, _, _, err := service.GetSSHUserAndServerByScpPath(selector)
		if err != nil || user.UserName != "ubuntu" {
			t.Fatalf("SCP override failed: %v", err)
		}
	}
	var unchanged model.Key
	if err := connection.First(&unchanged, "key_id = ?", "shared-key").Error; err != nil {
		t.Fatal(err)
	}
	if unchanged.UserName != "root" {
		t.Fatal("shared key username changed")
	}
	// Resolved identities must not mutate the underlying key request either.
	keys := []model.AddKeyRequest{{KeyID: tea.String("shared-key"), IdentityFile: tea.String("test.pem"), UserName: tea.String("root")}}
	_, err = service.GetSSHUsersByHost("192.0.2.1", map[string]model.Server{"192.0.2.1": {Host: "192.0.2.1", User: "ubuntu", KeyPairs: model.StringSlice{"shared-key"}}}, keys)
	if err != nil || *keys[0].UserName != "root" {
		t.Fatal("shared key request changed")
	}
}

func TestSSHSelectionPreservesExplicitUsersAndKeyAmbiguity(t *testing.T) {
	users := []model.SSHUser{{UserName: "root", KeyName: "one.pem"}, {UserName: "ubuntu", KeyName: "one.pem"}, {UserName: "root", KeyName: "two.pem"}}
	automatic, err := SelectSSHUser(users, "", "one.pem")
	if err != nil || !automatic.Auto || automatic.KeyName != "one.pem" {
		t.Fatal("automatic key selection failed")
	}
	explicit, err := SelectSSHUser(users, "ubuntu", "one.pem")
	if err != nil || explicit.Auto || explicit.UserName != "ubuntu" {
		t.Fatal("explicit selection changed")
	}
	if _, err := SelectSSHUser(users, "", ""); err == nil {
		t.Fatal("ambiguous keys were silently selected")
	}
	if _, err := SelectSSHUser(users, "ec2-user", "one.pem"); err == nil {
		t.Fatal("explicit unknown user was guessed")
	}
	if got := AutomaticSSHUsers(users); len(got) != 2 || !got[0].Auto {
		t.Fatal("automatic options were not grouped by key")
	}
}

func TestAutomaticSelectionDeduplicatesCredentialAliases(t *testing.T) {
	users := []model.SSHUser{{UserName: "root", KeyName: "root-label.pem", Base64Pem: "same-managed-credential"}, {UserName: "ubuntu", KeyName: "ubuntu-label.pem", Base64Pem: "same-managed-credential"}}
	if got := AutomaticSSHUsers(users); len(got) != 1 {
		t.Fatal("identical key aliases produced duplicate automatic choices")
	}
	selected, err := SelectSSHUser(users, "", "")
	if err != nil || !selected.Auto {
		t.Fatalf("identical key aliases were considered ambiguous: %v", err)
	}
}
