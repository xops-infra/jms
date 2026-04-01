package api

import (
	"errors"
	"net/http"
	"testing"
)

func TestPresentApprovalCreateErrorCreatePolicyConflict(t *testing.T) {
	status, message := presentApprovalCreateError("create_policy", errors.New("policy already exists"))
	if status != http.StatusConflict {
		t.Fatalf("unexpected status: got %d want %d", status, http.StatusConflict)
	}
	if message != "同名申请已存在，请修改申请名称后重试。" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestPresentApprovalCreateErrorMasksInternalFailure(t *testing.T) {
	status, message := presentApprovalCreateError("link_approval", errors.New(`ERROR: null value in column "users" of relation "jms_go_policy" violates not-null constraint (SQLSTATE 23502)`))
	if status != http.StatusInternalServerError {
		t.Fatalf("unexpected status: got %d want %d", status, http.StatusInternalServerError)
	}
	if message != "审批单已创建，但策略关联失败，请联系管理员处理。" {
		t.Fatalf("unexpected message: %q", message)
	}
}

func TestPresentApprovalCreateErrorDingtalkFailure(t *testing.T) {
	status, message := presentApprovalCreateError("create_dingtalk", errors.New("gateway timeout"))
	if status != http.StatusBadGateway {
		t.Fatalf("unexpected status: got %d want %d", status, http.StatusBadGateway)
	}
	if message != "提交申请失败，审批服务暂不可用，请稍后重试。" {
		t.Fatalf("unexpected message: %q", message)
	}
}
