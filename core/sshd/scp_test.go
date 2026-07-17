package sshd

import "testing"

func TestScpTarget(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    string
		wantErr bool
	}{
		{
			name: "upload without extra flags",
			args: []string{"-t", "root@10.193.18.8:/root/lw/"},
			want: "root@10.193.18.8:/root/lw/",
		},
		{
			name: "download without extra flags",
			args: []string{"-f", "root@10.193.18.8:/root/lw/pdf.zip"},
			want: "root@10.193.18.8:/root/lw/pdf.zip",
		},
		{
			// 复现线上问题：客户端加 -v 调试时 OpenSSH 注入 -v，
			// 旧实现固定取 args[1] 会得到 "-t"，导致 no IP address found。
			name: "upload with -v injected before -t",
			args: []string{"-v", "-t", "root@10.193.18.8#key_name=demo:/root/lw/"},
			want: "root@10.193.18.8#key_name=demo:/root/lw/",
		},
		{
			name: "download with -v injected before -f",
			args: []string{"-v", "-f", "root@10.193.18.8:/root/lw/pdf.zip"},
			want: "root@10.193.18.8:/root/lw/pdf.zip",
		},
		{
			name: "multiple flags before -t",
			args: []string{"-v", "-r", "-t", "root@10.193.18.8:/data/"},
			want: "root@10.193.18.8:/data/",
		},
		{
			name:    "missing target after -t",
			args:    []string{"-v", "-t"},
			wantErr: true,
		},
		{
			name:    "no scp mode flag",
			args:    []string{"-v", "root@10.193.18.8:/data/"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := scpTarget(tt.args)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("scpTarget(%v) expected error, got nil", tt.args)
				}
				return
			}
			if err != nil {
				t.Fatalf("scpTarget(%v) error = %v", tt.args, err)
			}
			if got != tt.want {
				t.Fatalf("scpTarget(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}
