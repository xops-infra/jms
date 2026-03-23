package api

import (
	"testing"

	"github.com/xops-infra/jms/model"
)

func TestSelectSSHUserPrefersRequestedKey(t *testing.T) {
	users := []model.SSHUser{
		{UserName: "root", KeyName: "default.pem"},
		{UserName: "root", KeyName: "openclaw.pem"},
	}

	selected, err := selectSSHUser(users, "root", "openclaw.pem")
	if err != nil {
		t.Fatalf("selectSSHUser returned error: %v", err)
	}
	if selected.KeyName != "openclaw.pem" {
		t.Fatalf("expected openclaw.pem, got %q", selected.KeyName)
	}
}

func TestSelectSSHUserErrorsWhenRequestedKeyMissing(t *testing.T) {
	users := []model.SSHUser{
		{UserName: "root", KeyName: "default.pem"},
	}

	_, err := selectSSHUser(users, "root", "openclaw.pem")
	if err == nil {
		t.Fatal("expected error when requested key is missing")
	}
	if got, want := err.Error(), "ssh user root with key openclaw.pem not found"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
