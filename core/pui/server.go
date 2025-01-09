package pui

import (
	"errors"
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
	"github.com/xops-infra/jms/core/sshd"
	. "github.com/xops-infra/jms/model"
)

func (ui *PUI) getServersMenuV2(sess *ssh.Session) ([]MenuItem, error) {
	timeStart := time.Now()
	defer func() {
		sshd.Info(fmt.Sprintf("GetServersMenuV2 cost: %s", time.Since(timeStart)), sess)
	}()
	menu := make([]MenuItem, 0)
	matchPolicies := app.App.Sshd.SshdIO.GetUserPolicys((*sess).User())
	sshd.Info(fmt.Sprintf("matchPolicies: %d", len(matchPolicies)), sess)
	servers, err := app.App.DBIo.LoadServer()
	if err != nil {
		return menu, err
	}
	// sshd.Info(fmt.Sprintf("servers: %d", len(servers)), sess)
	serversMap := servers.ToMap()

	user, err := app.App.DBIo.DescribeUser((*sess).User())
	if err != nil {
		return nil, err
	}

	for _, server := range servers {

		info := make(map[string]string, 0)
		for _, key := range server.KeyPairs {
			info[serverInfoKey] += " " + key
		}
		info[serverHost] = server.Host
		// for _, sshUser := range sshUsers {
		// 	info[serverUser] += fmt.Sprintf("%s: %s", sshUser.UserName, sshUser.KeyName)
		// }
		log.Debugf("info: %s", tea.Prettify(info))

		// if server.Proxy != nil {
		// 	info[serverProxy] = server.Proxy.Host
		// 	info[serverProxyUser] = server.Proxy.LoginUser
		// 	info[serverProxyKeyIdentityFile] = server.Proxy.SSHUsers.IdentityFile
		// }

		subMenu := MenuItem{
			Label:        fmt.Sprintf("%s\t[√]\t%s\t%s", server.ID, server.Host, server.Name),
			Info:         info,
			SubMenuTitle: fmt.Sprintf("%s '%s'", UserLoginLabel, server.Name),
			GetSubMenu:   ui.getServerSSHUsersMenu(server, serversMap),
		}
		// 判断机器权限进入不同菜单
		if !app.App.Sshd.SshdIO.MatchPolicy(user, Connect, server, matchPolicies, false) {
			subMenu.Label = fmt.Sprintf("%s\t[x]\t%s\t%s", server.ID, server.Host, server.Name)
			subMenu.SubMenuTitle = SelectServer
			subMenu.GetSubMenu = getServerApproveMenu(server)
		}
		menu = append(menu, subMenu)
		log.Debugf("subMenu: %v", tea.Prettify(subMenu))
	}
	return menu, nil
}

func getApproveMenu(policies []*Policy) []MenuItem {
	var menu []MenuItem
	for _, policy := range policies {
		menu = append(menu, MenuItem{
			Label:        fmt.Sprintf("%s\t[-]\t待审批工单\t(only admin can see)", policy.Name),
			SubMenuTitle: "Policy Summary",
			SelectedFunc: func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) (bool, error) {
				return false, nil
			},
			GetSubMenu: getApproveSubMenu(policy),
		})
	}
	return menu
}

func getApproveSubMenu(policy *Policy) func(int, MenuItem, *ssh.Session, []MenuItem) []MenuItem {
	return func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) []MenuItem {
		sshd.Info(tea.Prettify(policy), sess)
		var menu []MenuItem
		menu = append(menu, MenuItem{
			Label: "Approve",
			SelectedFunc: func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) (bool, error) {
				err := app.App.DBIo.ApprovePolicy(policy.Name, (*sess).User(), true)
				if err != nil {
					return false, err
				}
				sshd.Info("Approve Success", sess)
				return true, nil
			}})
		menu = append(menu, MenuItem{
			Label: "Reject",
			SelectedFunc: func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) (bool, error) {
				err := app.App.DBIo.ApprovePolicy(policy.Name, (*sess).User(), false)
				if err != nil {
					return false, err
				}
				sshd.Info("Reject Success", sess)
				return true, nil
			}})
		return menu
	}
}

// func sortMenu(menu []MenuItem) []MenuItem {
// 	for i := 0; i < len(menu); i++ {
// 		for j := i + 1; j < len(menu); j++ {
// 			if menu[i].Label > menu[j].Label {
// 				menu[i], menu[j] = menu[j], menu[i]
// 			}
// 		}
// 	}
// 	return menu
// }

// 判断权限在这里实现
func (ui *PUI) getServerSSHUsersMenu(server Server, serversMap map[string]Server) func(int, MenuItem, *ssh.Session, []MenuItem) []MenuItem {
	return func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) []MenuItem {

		var menu []MenuItem

		// 获取实时 keys
		keys, err := app.App.DBIo.InternalLoadKey()
		if err != nil {
			log.Errorf("get keys error: %s", err)
			sshd.ErrorInfo(err, sess)
			return menu
		}
		// sshd.Info(fmt.Sprintf("all server keys: %d", len(keys)), sess)

		users, err := app.App.Sshd.SshdIO.GetSSHUsersByHost(server.Host, serversMap, keys)
		if err != nil {
			log.Errorf("get ssh users error: %s", err)
			sshd.ErrorInfo(err, sess)
			return menu
		}

		for _, sshUser := range users {
			subMenu := MenuItem{}
			log.Debugf("server:%s user:%s", server.Host, sshUser.UserName)
			subMenu.SelectedFunc = func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) (bool, error) {
				if server.Status != model.InstanceStatusRunning {
					return false, fmt.Errorf("%s status %s, can not login", server.Host, strings.ToLower(string(server.Status)))
				}
				// 记录登录日志到数据库
				if app.App.Config.WithDB.Enable {
					err := app.App.DBIo.AddServerLoginRecord(&AddSshLoginRequest{
						TargetServer: tea.String(server.Host),
						InstanceID:   tea.String(server.ID),
						User:         tea.String((*sess).User()),
						Client:       tea.String((*sess).RemoteAddr().String()),
					})
					if err != nil {
						log.Errorf("create ssh login record error: %s", err)
					}
				}
				// 进入的时候标记超时暂停检查
				ui.pause()
				defer ui.resume()
				err := sshd.NewTerminal(server, sshUser, sess)
				if err != nil {
					return false, err
				}
				// 登录之后就会阻塞在这里，如果主动退出继续执行后续代码
				return true, nil
			}
			subMenu.Label = fmt.Sprintf("key:%s user:%s", sshUser.KeyName, sshUser.UserName)
			menu = append(menu, subMenu)
		}

		if len(menu) == 0 {
			menu = append(menu, MenuItem{
				Label: "该机器密钥没有被 JMS 托管，无法登录",
				SelectedFunc: func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) (bool, error) {
					return false, errors.New("pls check instance key")
				},
			})
		}
		return menu
	}
}

func getServerApproveMenu(server Server) func(int, MenuItem, *ssh.Session, []MenuItem) []MenuItem {
	return func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) []MenuItem {
		sshd.ErrorInfo(fmt.Errorf("no permission for %s,Please apply for permission", server.Name), sess)
		var menu []MenuItem
		menu = append(menu, MenuItem{
			Label: fmt.Sprintf("Only this server: %s", server.Host),
			Info: map[string]string{
				serverInfoKey: server.Name,
			},
			SubMenuTitle: SelectAction,
			GetSubMenu: getActionMenu(ServerFilterV1{
				IpAddr: []string{server.Host},
				Name:   []string{server.Name},
			}),
		})
		// serverTeam := server.Tags.GetTeam()
		// if serverTeam != nil {
		// 	// 申请机器所在组权限
		// 	menu = append(menu, MenuItem{
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

func getActionMenu(serverFilter ServerFilterV1) func(int, MenuItem, *ssh.Session, []MenuItem) []MenuItem {
	return func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) []MenuItem {
		sshd.Info("申请行为，包括连接，上传和下载文件的权限。", sess)
		var menu []MenuItem
		for key, value := range DefaultPolicies {
			menu = append(menu, MenuItem{
				Label:      fmt.Sprintf("申请 %s 权限", key),
				GetSubMenu: getExpireMenu(serverFilter, value),
			})
		}
		return menu
	}
}

func getExpireMenu(serverFilter ServerFilterV1, actions ArrayString) func(int, MenuItem, *ssh.Session, []MenuItem) []MenuItem {
	return func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) []MenuItem {
		var menu []MenuItem
		sshd.Info("选择策略生效周期，到期后会自动删除策略。", sess)
		for expiredKey, value := range ExpireTimes {
			menu = append(menu, MenuItem{
				Label:        fmt.Sprintf("策略有效期 %s", expiredKey),
				SubMenuTitle: "Summary",
				GetSubMenu:   getSureApplyMenu(serverFilter, actions, value),
			})
		}
		return menu
	}
}

func getSureApplyMenu(serverFilter ServerFilterV1, actions ArrayString, expiredDuration time.Duration) func(int, MenuItem, *ssh.Session, []MenuItem) []MenuItem {
	return func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) []MenuItem {
		expired := time.Now().Add(expiredDuration)
		policyNew := &PolicyRequest{
			Actions:        actions,
			Users:          ArrayString{(*sess).User()},
			ServerFilterV1: &serverFilter,
			ExpiresAt:      &expired,
			Name:           tea.String(fmt.Sprintf("%s-%s", (*sess).User(), time.Now().Format("20060102_1504"))),
		}
		var menu []MenuItem
		sshd.Info(fmt.Sprintf("%s\n确定申请权限？", tea.Prettify(policyNew)), sess)
		menu = append(menu, MenuItem{
			Label: "确定",
			SelectedFunc: func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) (bool, error) {
				log.Infof("create policy: %s", tea.Prettify(policyNew))
				if policyNew.ServerFilterV1 == nil {
					return false, fmt.Errorf("server filter is nil")
				}

				// 创建审批策略
				policyId, err := app.App.DBIo.CreatePolicy(policyNew)
				if err != nil {
					log.Errorf("create policy error: %s", err)
					return false, err
				}
				log.Infof("create approve success, id: %s", policyId)

				// 创建dingtalk审批
				if app.App.Config.WithDingtalk.Enable {
					id, err := dingtalk.CreateApproval((*sess).User(), []dt.FormComponentValue{
						{
							Name:  tea.String("EnvType"),
							Value: tea.String(FmtDingtalkApproveFile(policyNew.ServerFilterV1.EnvType)),
						},
						{
							Name:  tea.String("ServerFilter"),
							Value: tea.String(tea.Prettify(policyNew.ServerFilterV1)),
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
					// 更新审批ID到策略字段
					if err := app.App.DBIo.UpdatePolicy(policyId, &PolicyRequest{
						ApprovalID: &id,
					}); err != nil {
						log.Errorf("update policy approval id error, report to admin: %s", err)
						return false, err
					}

					sshd.Info(fmt.Sprintf("成功创建钉钉审批:%s 等等管理员审批 完成后策略自动生效", id), sess)
				}

				// 产生一个申请权限的任务，等待管理员审核
				sshd.Info(fmt.Sprintf("审批ID:%s，创建成功！等待管理员审核。", policyId), sess)
				return true, nil
			}})
		return menu
	}
}
