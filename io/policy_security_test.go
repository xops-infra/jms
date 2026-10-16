package io

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/jms/model"
	cloudmodel "github.com/xops-infra/multi-cloud-sdk/pkg/model"
	"github.com/xops-infra/noop/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExplicitDenyOverridesImplicitAccess(t *testing.T) {
	log.Default().WithLevel(log.ErrorLevel).WithFilename(filepath.Join(t.TempDir(), "test.log")).Init()
	service := NewSshd(db.NewJmsDbService(nil), nil)
	server := model.Server{ID: "target", Host: "192.0.2.1", Tags: cloudmodel.Tags{
		{Key: "Team", Value: "ops"}, {Key: "Owner", Value: "owner"},
	}}
	users := []model.User{
		{Username: tea.String("administrator"), Groups: model.ArrayString{"admin"}},
		{Username: tea.String("teammate"), Groups: model.ArrayString{"ops"}},
		{Username: tea.String("owner")},
	}
	for _, user := range users {
		for _, action := range []model.Action{model.Connect, model.Upload, model.Download} {
			t.Run(*user.Username+"/"+string(action), func(t *testing.T) {
				policy := model.Policy{IsEnabled: true, ExpiresAt: time.Now().Add(time.Hour),
					ServerFilterV1: &model.ServerFilterV1{ID: []string{server.ID}},
					Actions:        model.ArrayString{string(action), string(model.ReverseAction(action))},
				}
				if !service.MatchPolicy(user, action, server, nil, false) {
					t.Fatal("existing implicit access should remain without a deny")
				}
				if service.MatchPolicy(user, action, server, []model.Policy{policy}, false) {
					t.Fatal("explicit deny must override implicit access and a same-policy allow")
				}
				policy.ExpiresAt = time.Now().Add(-time.Hour)
				if !service.MatchPolicy(user, action, server, []model.Policy{policy}, false) {
					t.Fatal("expired deny should not affect access")
				}
				policy.ExpiresAt = time.Now().Add(time.Hour)
				policy.IsEnabled = false
				if !service.MatchPolicy(user, action, server, []model.Policy{policy}, false) {
					t.Fatal("disabled deny should not affect access")
				}
				policy.IsEnabled = true
				policy.IsDeleted = true
				if !service.MatchPolicy(user, action, server, []model.Policy{policy}, false) {
					t.Fatal("deleted deny should not affect access")
				}
			})
		}
	}
}

func TestWildcardGroupsDoNotGrantAdminAccess(t *testing.T) {
	log.Default().WithLevel(log.ErrorLevel).WithFilename(filepath.Join(t.TempDir(), "test.log")).Init()
	service := NewSshd(db.NewJmsDbService(nil), nil)
	for _, group := range []string{"*", "!other", "adm*"} {
		user := model.User{Username: tea.String("ordinary"), Groups: model.ArrayString{group}}
		if service.MatchPolicy(user, model.Connect, model.Server{}, nil, false) {
			t.Fatalf("group %q must not confer the admin role", group)
		}
	}
	user := model.User{Username: tea.String("deleted"), Groups: model.ArrayString{"admin"}, IsDeleted: tea.Bool(true)}
	if service.MatchPolicy(user, model.Connect, model.Server{}, nil, false) {
		t.Fatal("deleted user must not retain access")
	}
}

func TestFileTransferHonorsFullAssetFilter(t *testing.T) {
	log.Default().WithLevel(log.ErrorLevel).WithFilename(filepath.Join(t.TempDir(), "test.log")).Init()
	rdb, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "permissions.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := rdb.AutoMigrate(&model.Server{}, &model.Policy{}); err != nil {
		t.Fatal(err)
	}
	servers := []model.Server{
		{ID: "allowed", Name: "allowed-server", Host: "192.0.2.1", Tags: cloudmodel.Tags{{Key: "EnvType", Value: "dev"}, {Key: "Team", Value: "ops"}}},
		{ID: "denied", Name: "denied-server", Host: "192.0.2.2", Tags: cloudmodel.Tags{{Key: "EnvType", Value: "prod"}, {Key: "Team", Value: "other"}}},
	}
	if err := rdb.Create(&servers).Error; err != nil {
		t.Fatal(err)
	}
	service := NewSshd(db.NewJmsDbService(rdb), nil)
	user := model.User{Username: tea.String("ordinary")}
	filters := []*model.ServerFilterV1{
		{ID: []string{"allowed"}},
		{Name: []string{"allowed-server"}},
		{EnvType: []string{"dev"}},
		{Team: []string{"ops"}},
		{IpAddr: []string{"192.0.2.*"}, EnvType: []string{"dev"}},
	}
	for index, filter := range filters {
		if err := rdb.Where("1 = 1").Delete(&model.Policy{}).Error; err != nil {
			t.Fatal(err)
		}
		policy := model.Policy{ID: "transfer", Name: "transfer", Users: model.ArrayString{"ordinary"},
			Actions:   model.ArrayString{string(model.Upload), string(model.Download)},
			IsEnabled: true, ExpiresAt: time.Now().Add(time.Hour), ServerFilterV1: filter,
		}
		if err := rdb.Create(&policy).Error; err != nil {
			t.Fatal(err)
		}
		for _, action := range []model.Action{model.Upload, model.Download} {
			if err := service.CheckPermission("deploy@192.0.2.1:/tmp/file", user, action); err != nil {
				t.Fatalf("filter %d rejected allowed transfer: %v", index, err)
			}
			if err := service.CheckPermission("deploy@192.0.2.2:/tmp/file", user, action); err == nil {
				t.Fatalf("filter %d allowed transfer outside its asset scope", index)
			}
		}
	}
}

func TestFileTransferChecksActualHost(t *testing.T) {
	log.Default().WithLevel(log.ErrorLevel).WithFilename(filepath.Join(t.TempDir(), "test.log")).Init()
	rdb, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "permissions.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := rdb.AutoMigrate(&model.Server{}, &model.Policy{}); err != nil {
		t.Fatal(err)
	}
	servers := []model.Server{
		{ID: "allowed", Host: "192.0.2.1"},
		{ID: "denied", Host: "denied.example.test"},
		{ID: "allowed-dns", Host: "allowed.example.test"},
	}
	if err := rdb.Create(&servers).Error; err != nil {
		t.Fatal(err)
	}
	policy := model.Policy{ID: "transfer", Name: "transfer", Users: model.ArrayString{"ordinary"},
		Actions: model.ArrayString{string(model.Upload), string(model.Download)}, IsEnabled: true,
		ExpiresAt: time.Now().Add(time.Hour), ServerFilterV1: &model.ServerFilterV1{ID: []string{"allowed", "allowed-dns"}},
	}
	if err := rdb.Create(&policy).Error; err != nil {
		t.Fatal(err)
	}
	service := NewSshd(db.NewJmsDbService(rdb), nil)
	user := model.User{Username: tea.String("ordinary")}
	for _, action := range []model.Action{model.Upload, model.Download} {
		for _, target := range []string{
			"deploy@denied.example.test:/tmp/192.0.2.1",
			"deploy@denied.example.test#key_name=deploy.pem:/tmp/192.0.2.1",
		} {
			if err := service.CheckPermission(target, user, action); err == nil {
				t.Fatalf("allowed %s on a denied DNS host using an IP in the path: %s", action, target)
			}
		}
		for _, target := range []string{
			"deploy@allowed.example.test:/tmp/file",
			"deploy@allowed.example.test#key_name=deploy.pem:/tmp/file",
		} {
			if err := service.CheckPermission(target, user, action); err != nil {
				t.Fatalf("rejected allowed DNS transfer %s: %v", target, err)
			}
		}
	}
}

func TestPolicyLoadFailureBlocksImplicitTransferAccess(t *testing.T) {
	log.Default().WithLevel(log.ErrorLevel).WithFilename(filepath.Join(t.TempDir(), "test.log")).Init()
	rdb, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "permissions.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	// Leave the policy table absent while retaining a readable asset inventory.
	if err := rdb.AutoMigrate(&model.Server{}); err != nil {
		t.Fatal(err)
	}
	server := model.Server{ID: "target", Host: "192.0.2.1", Tags: cloudmodel.Tags{
		{Key: "Team", Value: "ops"}, {Key: "Owner", Value: "owner"},
	}}
	if err := rdb.Create(&server).Error; err != nil {
		t.Fatal(err)
	}
	service := NewSshd(db.NewJmsDbService(rdb), nil)
	if policies, err := service.GetUserPolicys("administrator"); err == nil || policies != nil {
		t.Fatalf("policy query failure must be returned without usable policies: policies=%v err=%v", policies, err)
	}
	for _, user := range []model.User{
		{Username: tea.String("administrator"), Groups: model.ArrayString{"admin"}},
		{Username: tea.String("teammate"), Groups: model.ArrayString{"ops"}},
		{Username: tea.String("owner")},
	} {
		for _, action := range []model.Action{model.Upload, model.Download} {
			if err := service.CheckPermission("deploy@192.0.2.1:/tmp/file", user, action); err == nil {
				t.Fatalf("policy load failure retained implicit %s access for %s", action, *user.Username)
			}
		}
	}
}

func TestPoliciesPreserveNonDatabaseAccess(t *testing.T) {
	service := NewSshd(nil, nil)
	policies, err := service.GetUserPolicys("legacy")
	if err != nil || len(policies) != 1 {
		t.Fatalf("legacy policy loading changed: policies=%v err=%v", policies, err)
	}
	user := model.User{Username: tea.String("legacy")}
	if !service.MatchPolicy(user, model.Connect, model.Server{}, policies, false) {
		t.Fatal("non-database access should retain the existing allow behavior")
	}
}
