package cmd

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xops-infra/jms/model"
)

func TestParseRunOptions(t *testing.T) {
	opts, err := parseRunOptions([]string{
		"--target", "prod-01", "--user", "deploy", "--timeout", "2m", "--",
		"printf", "%s", "hello world",
	})
	if err != nil {
		t.Fatalf("parseRunOptions() error = %v", err)
	}
	if opts.Target != "prod-01" || opts.User != "deploy" || opts.Timeout != 2*time.Minute {
		t.Fatalf("parseRunOptions() = %#v", opts)
	}
	wantCommand := []string{"printf", "%s", "hello world"}
	if !reflect.DeepEqual(opts.Command, wantCommand) {
		t.Fatalf("command = %#v, want %#v", opts.Command, wantCommand)
	}
}

func TestParseRunOptionsRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "missing target", args: []string{"--user", "deploy", "--", "true"}},
		{name: "missing user", args: []string{"--target", "prod-01", "--", "true"}},
		{name: "missing command", args: []string{"--target", "prod-01", "--user", "deploy"}},
		{name: "shell missing value", args: []string{"--target", "prod-01", "--user", "deploy", "--shell"}},
		{name: "base64 and argv", args: []string{"--target", "prod-01", "--user", "deploy", "--shell-base64", "dHJ1ZQ==", "false"}},
		{name: "timeout too small", args: []string{"--target", "prod-01", "--user", "deploy", "--timeout", "100ms", "--", "true"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := parseRunOptions(tt.args); err == nil {
				t.Fatal("parseRunOptions() expected an error")
			}
		})
	}
}

func TestParseRunOptionsShellConsumesRemainder(t *testing.T) {
	opts, err := parseRunOptions([]string{
		"--target", "prod-01", "--user", "deploy", "--shell",
		"printf", "stdout-ok;", "printf", "stderr-ok", ">&2;", "exit", "7",
	})
	if err != nil {
		t.Fatalf("parseRunOptions() error = %v", err)
	}
	want := "printf stdout-ok; printf stderr-ok >&2; exit 7"
	if opts.Shell != want {
		t.Fatalf("shell = %q, want %q", opts.Shell, want)
	}
}

func TestResolveExactServer(t *testing.T) {
	servers := model.Servers{
		{ID: "i-1", Name: "prod", Host: "10.0.0.1"},
		{ID: "i-2", Name: "prod", Host: "10.0.0.2"},
	}
	server, err := resolveExactServer(servers, "i-2")
	if err != nil || server.Host != "10.0.0.2" {
		t.Fatalf("resolveExactServer() = %#v, %v", server, err)
	}
	if _, err := resolveExactServer(servers, "prod"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("resolveExactServer() ambiguous error = %v", err)
	}
}

func TestResolveExactSSHUserRequiresKeyWhenAmbiguous(t *testing.T) {
	users := []model.SSHUser{
		{UserName: "deploy", KeyName: "key-a"},
		{UserName: "deploy", KeyName: "key-b"},
	}
	if _, err := resolveExactSSHUser(users, "deploy", ""); err == nil {
		t.Fatal("resolveExactSSHUser() expected ambiguity error")
	}
	user, err := resolveExactSSHUser(users, "deploy", "key-b")
	if err != nil || user.KeyName != "key-b" {
		t.Fatalf("resolveExactSSHUser() = %#v, %v", user, err)
	}
}

func TestJoinShellArgs(t *testing.T) {
	got := joinShellArgs([]string{"printf", "%s", "it's safe", ""})
	want := `'printf' '%s' 'it'"'"'s safe' ''`
	if got != want {
		t.Fatalf("joinShellArgs() = %q, want %q", got, want)
	}
}
