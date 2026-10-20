package cmd

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
)

type testCommandAuditor struct {
	startErr, finishErr error
	started, finished   int
	result              model.AddSSHCommandRecordRequest
}

func (a *testCommandAuditor) StartSSHCommandRecord(model.AddSSHCommandRecordRequest) (string, error) {
	a.started++
	return "audit-id", a.startErr
}
func (a *testCommandAuditor) FinishSSHCommandRecord(_ string, req model.AddSSHCommandRecordRequest) error {
	a.finished++
	a.result = req
	return a.finishErr
}

func TestAuditedCommandRejectsUnrecordedExecutionAndNeverRetries(t *testing.T) {
	log.Default().WithFilename(filepath.Join(t.TempDir(), "audit.log")).WithLevel(log.FatalLevel).Init()
	for _, tc := range []struct {
		name                string
		startErr, finishErr error
		wantCalls, wantCode int
	}{
		{"start unavailable", errors.New("database unavailable"), nil, 0, exitAuditFailure},
		{"completion unavailable", nil, errors.New("database unavailable"), 1, 7},
		{"recorded", nil, nil, 1, 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := &testCommandAuditor{startErr: tc.startErr, finishErr: tc.finishErr}
			calls := 0
			identity := model.SSHUser{UserName: "root"}
			code, err := auditedCommand(a, model.AddSSHCommandRecordRequest{User: "authenticated-user"}, func() (int, error) {
				calls++
				identity.UserName = "ubuntu"
				return 7, nil
			}, &identity)
			if code != tc.wantCode || calls != tc.wantCalls || a.finished != tc.wantCalls {
				t.Fatalf("code=%d calls=%d finished=%d", code, calls, a.finished)
			}
			if tc.startErr != nil && err == nil {
				t.Fatal("missing audit failure")
			}
			if tc.startErr == nil && (err != nil || a.result.User != "authenticated-user" || a.result.SSHUser != "ubuntu" || a.result.ExitCode != 7) {
				t.Fatalf("lost execution result: %+v %v", a.result, err)
			}
		})
	}
}

func TestAuditedCommandDistinguishesTargetExit68FromTimeout(t *testing.T) {
	for _, executionError := range []error{nil, errors.New("command timed out")} {
		a := &testCommandAuditor{}
		_, _ = auditedCommand(a, model.AddSSHCommandRecordRequest{}, func() (int, error) {
			return exitExecutionTimedOut, executionError
		}, &model.SSHUser{})
		if a.result.TimedOut != (executionError != nil) {
			t.Fatal("target exit status must not imply timeout")
		}
	}
}
