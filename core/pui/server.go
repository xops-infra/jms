package pui

import (
	"fmt"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/elfgzp/ssh"
	dt "github.com/xops-infra/go-dingtalk-sdk-wrapper"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/dingtalk"
	"github.com/xops-infra/jms/core/instance"
	"github.com/xops-infra/jms/core/sshd"
	. "github.com/xops-infra/jms/model"
)

func GetServersMenuV2(sess *ssh.Session, user User, timeout string) ([]*MenuItem, error) {
	timeStart := time.Now()
	defer log.Debugf("GetServersMenuV2 cost %s", time.Since(timeStart).String())
	menu := make([]*MenuItem, 0)
	servers := instance.GetServers()
	var matchPolicies []Policy
	if app.App.JmsDBService == nil {
		// 如果没有使用数据库，则默认都可见
		matchPolicies = append(matchPolicies, Policy{
			Actions:   All,
			IsEnabled: true,
			Users:     ArrayString{*user.Username},
		})
	} else {
		policies, err := app.App.JmsDBService.QueryPolicyByUser(*user.Username)
		if err != nil {
			log.Errorf("query policy error: %s", err)
			return nil, err
		}
		matchPolicies = policies
	}

	for _, server := range servers {
		// 默认都可见，连接的时候再判断是否允许
		info := make(map[string]string, 0)
		for _, key := range server.KeyPairs {
			info[serverInfoKey] += " " + *key
		}
		info[serverHost] = server.Host
		for _, sshUser := range server.SSHUsers {
			info[serverUser] += fmt.Sprintf("%s: %s", sshUser.UserName, sshUser.KeyName)
		}
		// if server.Proxy != nil {
		// 	info[serverProxy] = server.Proxy.Host
		// 	info[serverProxyUser] = server.Proxy.LoginUser
		// 	info[serverProxyKeyIdentityFile] = server.Proxy.SSHUsers.IdentityFile
		// }
		subMenu := &MenuItem{
			Label:        fmt.Sprintf("%s\t[√]\t%s\t%s", server.ID, server.Host, server.Name),
			Info:         info,
			SubMenuTitle: fmt.Sprintf("%s '%s'", UserLoginLabel, server.Name),
			GetSubMenu:   GetServerSSHUsersMenu(server, timeout, matchPolicies),
		}

		// 判断机器权限进入不同菜单
		if !MatchPolicy(user, Connect, server, matchPolicies) {
			subMenu.Label = fmt.Sprintf("%s\t[x]\t%s\t%s", server.ID, server.Host, server.Name)
			subMenu.SubMenuTitle = SelectServer
			subMenu.GetSubMenu = getServerApproveMenu(server)
		}

		menu = append(menu, subMenu)
	}
	// sort menu
	// menu = sortMenu(menu)
	return menu, nil
}

func GetApproveMenu(policies []*Policy) []*MenuItem {
	var menu []*MenuItem
	for _, policy := range policies {
		menu = append(menu, &MenuItem{
			Label:        fmt.Sprintf("%s\t[-]\t待审批工单\t(only admin can see)", policy.Name),
			SubMenuTitle: "Policy Summary",
			SelectedFunc: func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) (bool, error) {
				return false, nil
			},
			GetSubMenu: getApproveSubMenu(policy),
		})
	}
	return menu
}

func getApproveSubMenu(policy *Policy) func(int, *MenuItem, *ssh.Session, []*MenuItem) []*MenuItem {
	return func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) []*MenuItem {
		sshd.Info(tea.Prettify(policy), sess)
		var menu []*MenuItem
		menu = append(menu, &MenuItem{
			Label: "Approve",
			SelectedFunc: func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) (bool, error) {
				err := app.App.JmsDBService.ApprovePolicy(policy.Name, (*sess).User(), true)
				if err != nil {
					return false, err
				}
				sshd.Info("Approve Success", sess)
				return true, nil
			}})
		menu = append(menu, &MenuItem{
			Label: "Reject",
			SelectedFunc: func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) (bool, error) {
				err := app.App.JmsDBService.ApprovePolicy(policy.Name, (*sess).User(), false)
				if err != nil {
					return false, err
				}
				sshd.Info("Reject Success", sess)
				return true, nil
			}})
		return menu
	}
}

func sortMenu(menu []*MenuItem) []*MenuItem {
	for i := 0; i < len(menu); i++ {
		for j := i + 1; j < len(menu); j++ {
			if menu[i].Label > menu[j].Label {
				menu[i], menu[j] = menu[j], menu[i]
			}
		}
	}
	return menu
}

// 判断权限在这里实现
func GetServerSSHUsersMenu(server Server, timeout string, matchPolicies []Policy) func(int, *MenuItem, *ssh.Session, []*MenuItem) []*MenuItem {
	return func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) []*MenuItem {
		var menu []*MenuItem
		subMenu := &MenuItem{}
		for _, sshUser := range server.SSHUsers {
			log.Debugf("server:%s user:%s", server.Host, sshUser.UserName)
			subMenu.Label = fmt.Sprintf("key:%s user:%s", sshUser.KeyName, sshUser.UserName)
			subMenu.SelectedFunc = func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) (bool, error) {
				if server.Status != model.InstanceStatusRunning {
					return false, fmt.Errorf("%s status %s, can not login", server.Host, strings.ToLower(string(server.Status)))
				}
				// 记录登录日志到数据库
				if app.App.Config.WithDB.Enable {
					err := app.App.JmsDBService.AddServerLoginRecord(&AddSshLoginRequest{
						TargetServer: tea.String(server.Host),
						User:         tea.String((*sess).User()),
						Client:       tea.String((*sess).RemoteAddr().String()),
					})
					if err != nil {
						log.Errorf("create ssh login record error: %s", err)
					}
				}
				err := sshd.NewTerminal(server, sshUser, sess, timeout)
				if err != nil {
					return false, err
				}
				return true, nil
			}
		}

		menu = append(menu, subMenu)
		return menu
	}
}

func getServerApproveMenu(server Server) func(int, *MenuItem, *ssh.Session, []*MenuItem) []*MenuItem {
	return func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) []*MenuItem {
		sshd.ErrorInfo(fmt.Errorf("no permission for %s,Please apply for permission", server.Name), sess)
		var menu []*MenuItem
		menu = append(menu, &MenuItem{
			Label: fmt.Sprintf("Only this server: %s", server.Host),
			Info: map[string]string{
				serverInfoKey: server.Name,
			},
			SubMenuTitle: SelectAction,
			GetSubMenu: getActionMenu(ServerFilter{
				IpAddr: []string{server.Host},
			}),
		})
		// serverTeam := server.Tags.GetTeam()
		// if serverTeam != nil {
		// 	// 申请机器所在组权限
		// 	menu = append(menu, &MenuItem{
		// 		Label:        fmt.Sprintf("All Server with tag: Team=%s", *serverTeam),
		// 		Info:         map[string]string{},
		// 		SubMenuTitle: SelectAction,
		// 		GetSubMenu: getActionMenu(ServerFilter{
		// 			Team: serverTeam,
		// 		}),
		// 	})
		// }
		return menu
	}
}

func getActionMenu(serverFilter ServerFilter) func(int, *MenuItem, *ssh.Session, []*MenuItem) []*MenuItem {
	return func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) []*MenuItem {
		sshd.Info("申请行为，包括连接，上传和下载文件的权限。", sess)
		var menu []*MenuItem
		for key, value := range DefaultPolicies {
			menu = append(menu, &MenuItem{
				Label:      fmt.Sprintf("申请 %s 权限", key),
				GetSubMenu: getExpireMenu(serverFilter, value),
			})
		}
		return menu
	}
}

func getExpireMenu(serverFilter ServerFilter, actions ArrayString) func(int, *MenuItem, *ssh.Session, []*MenuItem) []*MenuItem {
	return func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) []*MenuItem {
		var menu []*MenuItem
		sshd.Info("选择策略生效周期，到期后会自动删除策略。", sess)
		for expiredKey, value := range ExpireTimes {
			menu = append(menu, &MenuItem{
				Label:        fmt.Sprintf("策略有效期 %s", expiredKey),
				SubMenuTitle: "Summary",
				GetSubMenu:   getSureApplyMenu(serverFilter, actions, value),
			})
		}
		return menu
	}
}

func getSureApplyMenu(serverFilter ServerFilter, actions ArrayString, expiredDuration time.Duration) func(int, *MenuItem, *ssh.Session, []*MenuItem) []*MenuItem {
	return func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) []*MenuItem {
		expired := time.Now().Add(expiredDuration)
		policyNew := &PolicyRequest{
			Actions:      actions,
			Users:        ArrayString{(*sess).User()},
			ServerFilter: &serverFilter,
			ExpiresAt:    &expired,
			Name:         tea.String(fmt.Sprintf("%s-%s", (*sess).User(), time.Now().Format("20060102_1504"))),
		}
		var menu []*MenuItem
		sshd.Info(fmt.Sprintf("%s\n确定申请权限？", tea.Prettify(policyNew)), sess)
		menu = append(menu, &MenuItem{
			Label: "确定",
			SelectedFunc: func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) (bool, error) {
				log.Infof("create policy: %s", tea.Prettify(policyNew))
				var approval_id string
				if policyNew.ServerFilter == nil {
					return false, fmt.Errorf("server filter is nil")
				}

				if app.App.Config.WithDingtalk.Enable {
					id, err := dingtalk.CreateApproval((*sess).User(), []dt.FormComponentValue{
						{
							Name:  tea.String("EnvType"),
							Value: tea.String(FmtDingtalkApproveFile(policyNew.ServerFilter.EnvType)),
						},
						{
							Name:  tea.String("ServerFilter"),
							Value: tea.String(tea.Prettify(policyNew.ServerFilter)),
						},
						{
							Name:  tea.String("DateExpired"),
							Value: tea.String(policyNew.ExpiresAt.Format(time.RFC3339)),
						},
						{
							Name:  tea.String("Actions"),
							Value: tea.String(tea.Prettify(policyNew.Actions)),
						},
						{
							Name:  tea.String("Comment"),
							Value: tea.String("来自jmsCli发起的策略申请"),
						},
					})
					if err != nil {
						log.Errorf("dingtalk.CreateApproval error: %s", err)
						return false, err
					}
					approval_id = id
					sshd.Info(fmt.Sprintf("成功创建钉钉审批:%s 等等管理员审批 完成后策略自动生效", id), sess)
				}
				policyId, err := app.App.JmsDBService.CreatePolicy(policyNew, &approval_id)
				if err != nil {
					log.Errorf("create policy error: %s", err)
					return false, err
				}
				log.Infof("create approve success, id: %s", policyId)
				// 产生一个申请权限的任务，等待管理员审核
				sshd.Info(fmt.Sprintf("审批ID:%s，创建成功！等待管理员审核。", policyId), sess)
				if false {
					// TODO: 发送钉钉消息
					instance.SendMessage(app.App.Config.WithSSHCheck.Alert.RobotToken,
						fmt.Sprintf("新增审批ID:%s，创建成功！等待管理员审核。\n%s", policyId, tea.Prettify(policyNew)))
				}
				return true, nil
			}})
		return menu
	}
}
