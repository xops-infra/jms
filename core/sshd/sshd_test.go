package sshd_test

import (
	"testing"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/sshd"
	"github.com/xops-infra/jms/model"
)

func init() {
	model.LoadYaml("/opt/jms/config.yaml")
	app.NewSshdApplication(true, "", "---").WithDB()
}

func TestAuditArch(t *testing.T) {
	sshd.AuditLogArchiver()
}
