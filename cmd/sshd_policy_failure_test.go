package cmd

import (
	"path/filepath"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/db"
	appio "github.com/xops-infra/jms/io"
	"github.com/xops-infra/jms/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestLoadCommandContextRejectsPolicyFailure(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "permissions.db")), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	// A valid user and inventory must not hide unavailable authorization policies.
	if err := database.AutoMigrate(&model.User{}, &model.Server{}); err != nil {
		t.Fatal(err)
	}
	user := model.User{ID: "administrator", Username: tea.String("administrator"), Groups: model.ArrayString{"admin"}}
	if err := database.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	previousApp := app.App
	app.App = &app.Application{Config: &model.Config{WithDB: model.WithPolicy{Enable: true}}, DBIo: db.NewJmsDbService(database)}
	app.App.Sshd.SshdIO = appio.NewSshd(app.App.DBIo, nil)
	t.Cleanup(func() {
		app.App = previousApp
		if sqlDB, err := database.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	if _, _, policies, err := loadCommandContext("administrator"); err == nil || policies != nil {
		t.Fatalf("command context must reject failed policy loading: policies=%v err=%v", policies, err)
	}
}
