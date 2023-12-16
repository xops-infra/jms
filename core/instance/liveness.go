package instance

import (
	"context"
	"fmt"
	"time"

	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/sshd"
	"github.com/xops-infra/jms/utils"
)

// dingtalkToken 为钉钉机器人的token
func ServerLiveness(dingtalkToken string) {
	timeStart := time.Now()
	servers := GetServers()
	for _, server := range servers {
		isIgnore := true
		for _, checkIp := range app.App.Config.WithSSHCheck.IPS {
			if checkIp == server.Host {
				isIgnore = false
				break
			}
		}

		if isIgnore {
			continue
		}

		for _, sshUser := range *server.SSHUsers {
			proxyClient, client, err := sshd.NewSSHClient(server, sshUser)
			if err != nil {
				_, found := app.App.Cache.Get(server.Host)
				if found {
					return
				}
				app.App.Cache.Add(server.Host, 1, 0)
				SendMessage(dingtalkToken, fmt.Sprintf("（紧急）机器ssh连接失败，请检查机器是否失联！\n机器名称：%s\n机器IP：%s\n登录用户：%s\n告警时间：%s\n错误信息：%s", server.Name, server.Host,
					sshUser.SSHUsername, time.Now().Format(time.RFC3339), err))
				continue
			}
			if proxyClient != nil {
				defer proxyClient.Close()
			}
			defer client.Close()

			if false {
				// TODO: 这里可以实现对目标机器的自定义命令检查
				sess, _ := client.NewSession()
				defer sess.Close()
				// 执行命令
				info, err := sess.Output(`netstat -an | grep ESTABLISHED | grep ":22" | wc -l`)
				if err != nil {
					log.Errorf("server liveness check error: %s", err)
				}
				log.Infof("server %s SSH ESTABLISHED nu: %s ", server.Host, string(info))
			}

			_, found := app.App.Cache.Get(server.Host)
			if found {
				SendMessage(dingtalkToken, fmt.Sprintf("机器ssh连接已经恢复！\n机器名称：%s\n机器IP：%s\n告警时间：%s\n登录用户：%s", server.Name, server.Host, time.Now().Format(time.RFC3339), sshUser.SSHUsername))
				app.App.Cache.Delete(server.Host)
			}
		}
	}
	log.Infof("server liveness check done cost: %s", time.Since(timeStart))
}

// 发送到群里
func SendMessage(token, msg string) {
	err := app.App.DT.SendMessage(context.Background(), &utils.SendMessageRequest{
		AccessToken: token,
		MessageContent: utils.MessageContent{
			MsgType: "text",
			Text: utils.TextBody{
				Content: msg,
			},
		},
	})
	if err != nil {
		log.Errorf("send dingtalk msg error: %s", err)
	}
	log.Infof("send dingtalk msg: %s", msg)
}
