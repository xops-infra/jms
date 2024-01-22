package sshd_test

import (
	"testing"

	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/core/sshd"
)

func init() {
	log.Default().Init()
	config.Load("/opt/jms/.jms.yml")
	app.NewSshdApplication(true, "~/.ssh/").WithPolicy()
}

func TestAuditArch(t *testing.T) {
	sshd.AuditLogArchiver()
}
