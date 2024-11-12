package sshd_test

import (
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/sshd"
	"github.com/xops-infra/jms/model"
)

func init() {
	model.LoadYaml("/opt/jms/config.yaml")
	app.NewApp(true, "", "---")
}

func TestAuditArch(t *testing.T) {
	sshd.AuditLogArchiver()
}

// test CheckPermission
func TestCheckPermission(t *testing.T) {
	policy := model.Policy{
		IsEnabled: true,
		ServerFilterV1: &model.ServerFilterV1{
			IpAddr: []string{"10.9.0.1"},
		},
		Users:     []string{"zhoushoujian"},
		Actions:   model.ConnectOnly,
		ExpiresAt: time.Now().AddDate(0, 0, 1),
	}

	user := model.User{Username: tea.String("zhoushoujian")}

	err := sshd.CheckPermission("root@10.9.0.1:/data/xx.zip", user, model.Upload, []model.Policy{policy})

	if err != nil {
		t.Log("ok", err)
	} else {
		t.Error("shoud be error")
	}

	policy.Actions = model.DownloadOnly
	err = sshd.CheckPermission("root@10.9.0.1:/data/xx.zip", user, model.Upload, []model.Policy{policy})
	if err != nil {
		t.Log("ok", err)
	} else {
		t.Error("shoud be error")
	}

	err = sshd.CheckPermission("root@10.9.0.1:/data/xx.zip", user, model.Download, []model.Policy{policy})
	if err == nil {
		t.Log("ok", err)
	} else {
		t.Error("shoud be ok")
	}

	policy.Actions = model.UploadOnly
	err = sshd.CheckPermission("root@10.9.0.1:/data/xx.zip", user, model.Download, []model.Policy{policy})
	if err != nil {
		t.Log("ok", err)
	} else {
		t.Error("shoud be error")
	}

}
