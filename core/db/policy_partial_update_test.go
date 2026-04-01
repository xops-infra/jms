package db

import (
	"fmt"
	"testing"
	"time"

	"github.com/xops-infra/jms/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newPolicyTestService(t *testing.T) *DBService {
	t.Helper()

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	if err := db.AutoMigrate(&model.Policy{}); err != nil {
		t.Fatalf("migrate policy schema: %v", err)
	}
	return NewJmsDbService(db)
}

func seedPolicy(t *testing.T, service *DBService) model.Policy {
	t.Helper()

	policy := model.Policy{
		ID:        "policy-1",
		Name:      "liujia-20260331_0550",
		Users:     model.ArrayString{"liujia"},
		Actions:   model.ArrayString{string(model.Connect)},
		ExpiresAt: time.Now().Add(time.Hour),
		ServerFilterV1: &model.ServerFilterV1{
			IpAddr: []string{"10.0.0.1"},
		},
	}
	if err := service.DB.Create(&policy).Error; err != nil {
		t.Fatalf("seed policy: %v", err)
	}
	return policy
}

func TestUpdatePolicyApprovalIDDoesNotClearUsers(t *testing.T) {
	service := newPolicyTestService(t)
	original := seedPolicy(t, service)

	approvalID := "approval-123"
	if err := service.UpdatePolicy(original.ID, &model.PolicyRequest{
		ApprovalID: &approvalID,
	}); err != nil {
		t.Fatalf("update approval id: %v", err)
	}

	updated, err := service.QueryPolicyById(original.ID)
	if err != nil {
		t.Fatalf("query policy: %v", err)
	}

	if updated.ApprovalID != approvalID {
		t.Fatalf("unexpected approval id: got %q want %q", updated.ApprovalID, approvalID)
	}
	if len(updated.Users) != 1 || updated.Users[0] != original.Users[0] {
		t.Fatalf("users changed unexpectedly: got %#v want %#v", updated.Users, original.Users)
	}
	if len(updated.Actions) != 1 || updated.Actions[0] != original.Actions[0] {
		t.Fatalf("actions changed unexpectedly: got %#v want %#v", updated.Actions, original.Actions)
	}
	if updated.ServerFilterV1 == nil || len(updated.ServerFilterV1.IpAddr) != 1 || updated.ServerFilterV1.IpAddr[0] != original.ServerFilterV1.IpAddr[0] {
		t.Fatalf("server filter changed unexpectedly: got %#v want %#v", updated.ServerFilterV1, original.ServerFilterV1)
	}
}

func TestUpdatePolicyServerFilterOnlyDoesNotClearUsers(t *testing.T) {
	service := newPolicyTestService(t)
	original := seedPolicy(t, service)

	nextFilter := &model.ServerFilterV1{
		IpAddr: []string{"10.0.0.2"},
	}
	if err := service.UpdatePolicy(original.ID, &model.PolicyRequest{
		ServerFilterV1: nextFilter,
	}); err != nil {
		t.Fatalf("update server filter: %v", err)
	}

	updated, err := service.QueryPolicyById(original.ID)
	if err != nil {
		t.Fatalf("query policy: %v", err)
	}

	if updated.ServerFilterV1 == nil || len(updated.ServerFilterV1.IpAddr) != 1 || updated.ServerFilterV1.IpAddr[0] != nextFilter.IpAddr[0] {
		t.Fatalf("server filter not updated: got %#v want %#v", updated.ServerFilterV1, nextFilter)
	}
	if len(updated.Users) != 1 || updated.Users[0] != original.Users[0] {
		t.Fatalf("users changed unexpectedly: got %#v want %#v", updated.Users, original.Users)
	}
	if len(updated.Actions) != 1 || updated.Actions[0] != original.Actions[0] {
		t.Fatalf("actions changed unexpectedly: got %#v want %#v", updated.Actions, original.Actions)
	}
}
