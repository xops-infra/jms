package db

import (
	"github.com/xops-infra/jms/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path/filepath"
	"testing"
)

func TestSSHLoginHintScopeAndConditionalInvalidation(t *testing.T) {
	connection, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "hint.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.AutoMigrate(&model.SSHLoginHint{}); err != nil {
		t.Fatal(err)
	}
	service := NewJmsDbService(connection)
	if err := service.RememberSSHLogin("asset", "host:22", "fingerprint-one", "ubuntu"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ asset, address, fingerprint, want string }{{"asset", "host:22", "fingerprint-one", "ubuntu"}, {"other", "host:22", "fingerprint-one", ""}, {"asset", "new:22", "fingerprint-one", ""}, {"asset", "host:22", "fingerprint-two", ""}} {
		got, err := service.SSHLoginUsername(tc.asset, tc.address, tc.fingerprint)
		if err != nil || got != tc.want {
			t.Fatalf("unexpected scoped hint: %q %v", got, err)
		}
	}
	if err := service.RememberSSHLogin("asset", "host:22", "fingerprint-one", "ec2-user"); err != nil {
		t.Fatal(err)
	}
	if err := service.ForgetSSHLogin("asset", "host:22", "fingerprint-one", "ubuntu"); err != nil {
		t.Fatal(err)
	}
	got, err := service.SSHLoginUsername("asset", "host:22", "fingerprint-one")
	if err != nil || got != "ec2-user" {
		t.Fatal("invalidation deleted a newer successful identity")
	}
	if err := service.ForgetSSHLogin("asset", "host:22", "fingerprint-one", "ec2-user"); err != nil {
		t.Fatal(err)
	}
	got, err = service.SSHLoginUsername("asset", "host:22", "fingerprint-one")
	if err != nil || got != "" {
		t.Fatal("stale hint was not removed")
	}
}
