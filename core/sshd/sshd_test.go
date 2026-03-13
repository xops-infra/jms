package sshd

import "testing"

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
