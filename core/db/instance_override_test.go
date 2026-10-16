package db

import (
	"github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"path/filepath"
	"testing"
)

func TestAssetSyncPreservesKeyOnlyUsernameByAssetID(t *testing.T) {
	log.Default().WithFilename(filepath.Join(t.TempDir(), "sync.log")).WithLevel(log.FatalLevel).Init()
	connection, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "sync.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.AutoMigrate(&model.Server{}); err != nil {
		t.Fatal(err)
	}
	service := NewJmsDbService(connection)
	initial := []model.Server{{ID: "key-only", Host: "192.0.2.1", User: "ubuntu"}, {ID: "password", Host: "192.0.2.2", User: "custom", Passwd: "test-password"}, {ID: "old", Host: "192.0.2.3", User: "old-user"}}
	if err := connection.Create(&initial).Error; err != nil {
		t.Fatal(err)
	}
	fresh := []model.Server{{ID: "key-only", Host: "192.0.2.4", Name: "updated"}, {ID: "password", Host: "192.0.2.2"}, {ID: "replacement", Host: "192.0.2.3"}, {ID: "default", Host: "192.0.2.5"}}
	if err := service.UpdateServerWithDelete(fresh); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct{ id, user, passwd string }{{"key-only", "ubuntu", ""}, {"password", "custom", "test-password"}, {"replacement", "", ""}, {"default", "", ""}} {
		var got model.Server
		if err := connection.First(&got, "id = ?", tc.id).Error; err != nil {
			t.Fatal(err)
		}
		if got.User != tc.user || got.Passwd != tc.passwd {
			t.Fatalf("override not preserved correctly for %s", tc.id)
		}
		if tc.id == "key-only" && (got.Name != "updated" || got.Host != "192.0.2.4") {
			t.Fatal("cloud fields did not refresh")
		}
	}
}
