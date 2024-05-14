package sshd_test

import (
	"testing"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/core/sshd"
)

func init() {
	config.LoadYaml("/opt/jms/config.yaml")
	app.NewSshdApplication(true, "", "---").WithDB()
}

func TestAuditArch(t *testing.T) {
	sshd.AuditLogArchiver()
}
