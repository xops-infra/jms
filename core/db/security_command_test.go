package db

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/xops-infra/jms/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestCommandAuditLifecyclePreservesStartAndActualIdentity(t *testing.T) {
	connection, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "audit.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.AutoMigrate(&model.SSHCommandRecord{}); err != nil {
		t.Fatal(err)
	}
	service := NewJmsDbService(connection)
	req := model.AddSSHCommandRecordRequest{User: "operator", Command: "true", KeyFingerprint: "fingerprint", TargetServer: "192.0.2.2"}
	id, err := service.StartSSHCommandRecord(req)
	if err != nil {
		t.Fatal(err)
	}
	var row model.SSHCommandRecord
	if err := connection.First(&row, "uuid = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != "running" || row.User != "operator" || row.Command != "true" || row.Source != model.AuditSourceCLI {
		t.Fatalf("start: %+v", row)
	}
	req.SSHUser, req.SSHKey, req.ExitCode, req.TimedOut, req.Duration = "ubuntu", "example.pem", 68, true, 2*time.Second
	if err := service.FinishSSHCommandRecord(id, req); err != nil {
		t.Fatal(err)
	}
	if err := connection.First(&row, "uuid = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if row.Status != "completed" || row.ExitCode != 68 || !row.TimedOut || row.DurationMS != 2000 || row.SSHUser != "ubuntu" || row.KeyFingerprint != "fingerprint" {
		t.Fatalf("finish: %+v", row)
	}
	if err := service.FinishSSHCommandRecord(id, req); err == nil {
		t.Fatal("must not overwrite completed audit")
	}
	if err := service.FinishSSHCommandRecord("missing", req); err == nil {
		t.Fatal("missing audit must fail")
	}
	// GORM struct updates can silently omit zero values. Successful completion must retain them.
	id, err = service.StartSSHCommandRecord(req)
	if err != nil {
		t.Fatal(err)
	}
	req.ExitCode, req.TimedOut, req.Duration = 0, false, 0
	if err := service.FinishSSHCommandRecord(id, req); err != nil {
		t.Fatal(err)
	}
	var success model.SSHCommandRecord
	if err := connection.First(&success, "uuid = ?", id).Error; err != nil {
		t.Fatal(err)
	}
	if success.Status != "completed" || success.ExitCode != 0 || success.TimedOut || success.DurationMS != 0 {
		t.Fatalf("zero result lost: %+v", success)
	}
}

func TestAuditMigrationPreservesHistoricalUnknownSources(t *testing.T) {
	connection, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "legacy.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// Existing schema before source/target identity and lifecycle columns were added.
	if err := connection.Exec(`CREATE TABLE record_ssh_login (
 id integer PRIMARY KEY AUTOINCREMENT, created_at datetime, updated_at datetime, deleted_at datetime,
 user varchar(255) NOT NULL, client varchar(255) NOT NULL, target varchar(255) NOT NULL, target_instance_id varchar(255))`).Error; err != nil {
		t.Fatal(err)
	}
	if err := connection.Exec(`INSERT INTO record_ssh_login (created_at,user,client,target) VALUES (?, ?, ?, ?)`, time.Now(), "historical-user", "192.0.2.1", "192.0.2.2").Error; err != nil {
		t.Fatal(err)
	}
	if err := connection.AutoMigrate(&model.SSHLoginRecord{}); err != nil {
		t.Fatal(err)
	}
	service := NewJmsDbService(connection)
	rows, total, err := service.ListServerLoginRecord(model.QueryLoginRequest{Source: "unknown"})
	if err != nil || total != 1 || len(rows) != 1 || rows[0].Source != "" || rows[0].User != "historical-user" {
		t.Fatalf("history lost or misclassified: %+v %d %v", rows, total, err)
	}
	if err := connection.AutoMigrate(&model.SSHCommandRecord{}); err != nil {
		t.Fatal(err)
	}
	legacy := model.SSHCommandRecord{UUID: "historical-command", User: "historical-user", SSHUser: "ubuntu", Command: "true", ExitCode: 7}
	if err := connection.Create(&legacy).Error; err != nil {
		t.Fatal(err)
	}
	for _, column := range []string{"status", "source"} {
		if err := connection.Migrator().DropColumn(&model.SSHCommandRecord{}, column); err != nil {
			t.Fatal(err)
		}
	}
	if err := connection.AutoMigrate(&model.SSHCommandRecord{}); err != nil {
		t.Fatal(err)
	}
	commands, total, err := service.ListSSHCommandRecords(model.QuerySSHCommandRequest{User: "historical-user"})
	if err != nil || total != 1 || len(commands) != 1 || commands[0].Status != "" || commands[0].ExitCode != 7 || commands[0].Command != "true" {
		t.Fatalf("command history changed: %+v %v", commands, err)
	}
}
