package sshd

import (
	"strings"
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/model"
)

func TestProxyClientConfigErrorsDoNotExposeSecrets(t *testing.T) {
	const secret = "proxy-secret-must-not-appear"
	tests := []struct {
		name  string
		proxy model.CreateProxyRequest
		want  string
	}{
		{
			name: "invalid config with password",
			proxy: model.CreateProxyRequest{
				Host:        tea.String("proxy.example.invalid"),
				Port:        tea.Int(22),
				LoginPasswd: tea.String(secret),
			},
			want: "proxy config error",
		},
		{
			name: "missing authentication",
			proxy: model.CreateProxyRequest{
				Name:      tea.String(secret),
				Host:      tea.String("proxy.example.invalid"),
				Port:      tea.Int(22),
				LoginUser: tea.String("proxy-user"),
			},
			want: "proxy config error has no auth",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			proxyClient, client, err := ProxyClient(model.Server{}, tt.proxy, model.SSHUser{})
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected %q error, got %v", tt.want, err)
			}
			if proxyClient != nil || client != nil {
				t.Fatal("invalid configuration returned an SSH client")
			}
			if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "login_passwd") {
				t.Fatalf("configuration error contains secret fields: %v", err)
			}
			if !strings.Contains(err.Error(), "proxy.example.invalid") {
				t.Fatalf("configuration error lacks safe proxy host context: %v", err)
			}
		})
	}
}

func TestParseRawCommand(t *testing.T) {
	tests := []struct {
		name    string
		command string
		wantCmd string
		wantArg []string
		wantErr bool
	}{
		{
			name:    "empty command",
			command: "   ",
			wantErr: true,
		},
		{
			name:    "single command",
			command: "ssh",
			wantCmd: "ssh",
			wantArg: []string{},
		},
		{
			name:    "scp path with spaces in quotes",
			command: `scp -t "root@1.1.1.1:/tmp/a b.txt"`,
			wantCmd: "scp",
			wantArg: []string{"-t", "root@1.1.1.1:/tmp/a b.txt"},
		},
		{
			name:    "command with escaped quote",
			command: `exec "ssh-rsa AAAA\"test comment"`,
			wantCmd: "exec",
			wantArg: []string{"ssh-rsa AAAA\"test comment"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCmd, gotArg, err := ParseRawCommand(tt.command)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("ParseRawCommand() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRawCommand() error = %v", err)
			}
			if gotCmd != tt.wantCmd {
				t.Fatalf("ParseRawCommand() cmd = %q, want %q", gotCmd, tt.wantCmd)
			}
			if len(gotArg) != len(tt.wantArg) {
				t.Fatalf("ParseRawCommand() args len = %d, want %d (%v)", len(gotArg), len(tt.wantArg), gotArg)
			}
			for i := range gotArg {
				if gotArg[i] != tt.wantArg[i] {
					t.Fatalf("ParseRawCommand() args[%d] = %q, want %q", i, gotArg[i], tt.wantArg[i])
				}
			}
		})
	}
}
