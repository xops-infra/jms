package instance

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	go_dingtalk_sdk_wrapper "github.com/patsnapops/go-dingtalk-sdk-wrapper"
	"github.com/patsnapops/noop/log"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/sshd"
)

func ServerLiveness() {
	timeStart := time.Now()
	for _, server := range *app.App.Config.Servers {
		isIgnore := true
		for _, checkIp := range app.App.Config.Monitor.IPS {
			if checkIp == server.Host {
				isIgnore = false
				break
			}
		}

		if isIgnore {
			continue
		}

		for _, sshUser := range *server.SSHUsers {
			client, err := sshd.NewSSHClient(server, sshUser)
			if err != nil {
				_, found := app.App.Cache.Get(server.Host)
				if found {
					return
				}
				app.App.Cache.Add(server.Host, 1, 0)
				sendN(fmt.Sprintf("（紧急）机器ssh连接失败，请检查机器是否失联！\n机器名称：%s\n机器IP：%s\n登录用户：%s\n告警时间：%s\n错误信息：%s", server.Name, server.Host,
					sshUser.SSHUsername, time.Now().Format(time.RFC3339), err))
			} else {
				defer client.Close()
				_, found := app.App.Cache.Get(server.Host)
				if found {
					sendN(fmt.Sprintf("机器ssh连接已经恢复！\n机器名称：%s\n机器IP：%s\n告警时间：%s\n登录用户：%s", server.Name, server.Host, time.Now().Format(time.RFC3339), sshUser.SSHUsername))
					app.App.Cache.Delete(server.Host)
				}
			}
		}
	}
	log.Infof("server liveness check done cost: %s", time.Since(timeStart))
}

func sendN(msg string) {
	err := app.App.DT.MiniProgram.SendWorkNotification(context.Background(), &go_dingtalk_sdk_wrapper.SendWorkNotificationRequest{
		AgentId:    &app.App.Config.DingTalk.AgentId,
		UseridList: tea.String(strings.Join(app.App.Config.DingTalk.WorkNotificationUsers, ",")),
		ToAllUser:  tea.Bool(false),
		Msg: &go_dingtalk_sdk_wrapper.MessageContent{
			MsgType: "text",
			Text: go_dingtalk_sdk_wrapper.TextBody{
				Content: msg,
			},
		},
	}, app.App.DT.AccessToken.Token)
	if err != nil {
		log.Errorf("send dingtalk msg error: %s", err)
	}
	log.Infof("send dingtalk msg: %s", msg)
}
