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
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/jms/core/dingtalk"
	"github.com/xops-infra/jms/core/instance"
	"github.com/xops-infra/jms/core/sshd"
)

func GetServersMenuV2(sess *ssh.Session, user db.User, timeout string) []*MenuItem {
	timeStart := time.Now()
	defer log.Debugf("GetServersMenuV2 cost %s", time.Since(timeStart).String())
	menu := make([]*MenuItem, 0)
	servers := instance.GetServers()
	var matchPolicies []db.Policy
	if app.App.DBService == nil {
		// 如果没有使用数据库，则默认都可见
		matchPolicies = append(matchPolicies, db.Policy{
			Actions:   db.All,
			IsEnabled: tea.Bool(true),
			Users:     db.ArrayString{tea.String(*user.Username)},
		})
	} else {
		policies, err := app.App.DBService.QueryPolicyByUser(*user.Username)
		if err != nil {
			log.Errorf("query policy error: %s", err)
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
			info[serverUser] += fmt.Sprintf("%s: %s", sshUser.SSHUsername, sshUser.KeyName)
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
		if !matchPolicy(user, db.Connect, server, matchPolicies) {
			subMenu.Label = fmt.Sprintf("%s\t[x]\t%s\t%s", server.ID, server.Host, server.Name)
			subMenu.SubMenuTitle = SelectServer
			subMenu.GetSubMenu = getServerApproveMenu(server)
		}

		menu = append(menu, subMenu)
	}
	// sort menu
	// menu = sortMenu(menu)
	return menu
}

func GetApproveMenu(policies []*db.Policy) []*MenuItem {
	var menu []*MenuItem
	for _, policy := range policies {
		menu = append(menu, &MenuItem{
			Label:        fmt.Sprintf("%s\t[-]\t待审批工单\t(only admin can see)", *policy.Name),
			SubMenuTitle: "Policy Summary",
			SelectedFunc: func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) (bool, error) {
				return false, nil
			},
			GetSubMenu: getApproveSubMenu(policy),
		})
	}
	return menu
}

func getApproveSubMenu(policy *db.Policy) func(int, *MenuItem, *ssh.Session, []*MenuItem) []*MenuItem {
	return func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) []*MenuItem {
		sshd.Info(tea.Prettify(policy), sess)
		var menu []*MenuItem
		menu = append(menu, &MenuItem{
			Label: "Approve",
			SelectedFunc: func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) (bool, error) {
				err := app.App.DBService.ApprovePolicy(*policy.Name, (*sess).User(), true)
				if err != nil {
					return false, err
				}
				sshd.Info("Approve Success", sess)
				return true, nil
			}})
		menu = append(menu, &MenuItem{
			Label: "Reject",
			SelectedFunc: func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) (bool, error) {
				err := app.App.DBService.ApprovePolicy(*policy.Name, (*sess).User(), false)
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

// 连接，上传，下载的时候，需要根据policy来判断是否允许
func matchPolicy(user db.User, inPutAction db.Action, server config.Server, dbPolicies []db.Policy) bool {
	// 是否启用策略
	if !app.App.Config.WithPolicy.Enable {
		return true
	}
	// 默认策略优先判断
	if matchPolicyOwner(user, server) {
		return true
	}
	// 用户组一致则有权限
	if matchUserGroup(user, server) {
		return true
	}

	// 再去匹配策略
	for _, dbPolicy := range dbPolicies {
		if dbPolicy.IsEnabled == nil || !*dbPolicy.IsEnabled {
			continue
		}
		// Check server filter first. If there is no match, continue to next policy.
		if dbPolicy.ServerFilter != nil {
			if !MatchServer(*dbPolicy.ServerFilter, server) {
				continue
			}
		}
		if dbPolicy.Actions == nil || len(dbPolicy.Actions) == 0 {
			log.Errorf("policy %s actions is nil?", *dbPolicy.Name)
			continue
		}
		// Check deny actions first. If there is a match, return false.
		for _, action := range dbPolicy.Actions {
			if action == string(db.DenyConnect) && inPutAction == db.Connect {
				return false
			}
			if action == string(db.DenyDownload) && inPutAction == db.Download {
				return false
			}
			if action == string(db.DenyUpload) && inPutAction == db.Upload {
				return false
			}
		}
		// Check allow actions next. If there is a match, return true.
		for _, action := range dbPolicy.Actions {
			if action == string(db.Connect) && inPutAction == db.Connect {
				return true
			}
			if action == string(db.Download) && inPutAction == db.Download {
				return true
			}
			if action == string(db.Upload) && inPutAction == db.Upload {
				return true
			}
		}
	}
	// Default to not allowed
	return false
}

// Owner和用户一样则有权限
func matchPolicyOwner(user db.User, server config.Server) bool {
	if server.Tags.GetOwner() != nil && *server.Tags.GetOwner() == *user.Username {
		return true
	}
	return false
}

// 用户组一致则有权限
// admin有所有权限
func matchUserGroup(user db.User, server config.Server) bool {
	if user.Groups != nil {
		if user.Groups.Contains("admin") {
			return true
		}
		if server.Tags.GetTeam() != nil {
			for _, group := range user.Groups {
				if *server.Tags.GetTeam() == group.(string) {
					return true
				}
			}
		} else {
			return false
		}

	}
	return false
}

// 支持!开头的反向匹配
// 默认没有匹配到标签的允许访问
func MatchServer(filter db.ServerFilter, server config.Server) bool {
	if filter.Name != nil {
		if *filter.Name == "*" || *filter.Name == server.Name {
			return true
		}
	}
	if filter.IpAddr != nil {
		if *filter.IpAddr == "*" || *filter.IpAddr == server.Host {
			return true
		}
	}
	if filter.EnvType != nil {
		if server.Tags.GetEnvType() == nil {
			return true
		}
		if strings.HasPrefix(*filter.EnvType, "!") {
			if strings.TrimPrefix(*filter.EnvType, "!") == *server.Tags.GetEnvType() {
				return false
			} else {
				return true
			}
		}
		if *filter.EnvType == "*" || *filter.EnvType == *server.Tags.GetEnvType() {
			return true
		}
	}
	if filter.Team != nil {
		if server.Tags.GetTeam() == nil {
			return true
		}
		if *filter.Team == "*" || *filter.Team == *server.Tags.GetTeam() {
			return true
		}
	}

	return false
}

func GetServerSSHUsersMenu(server config.Server, timeout string, matchPolicies []db.Policy) func(int, *MenuItem, *ssh.Session, []*MenuItem) []*MenuItem {
	return func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) []*MenuItem {
		var menu []*MenuItem
		subMenu := &MenuItem{}
		for _, sshUser := range server.SSHUsers {
			log.Debugf("server:%s user:%s", server.Host, sshUser.SSHUsername)
			subMenu.Label = fmt.Sprintf("key:%s user:%s", sshUser.KeyName, sshUser.SSHUsername)
			subMenu.SelectedFunc = func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) (bool, error) {
				if server.Status != model.InstanceStatusRunning {
					return false, fmt.Errorf("%s status %s, can not login", server.Host, strings.ToLower(string(server.Status)))
				}
				// 记录登录日志到数据库
				if app.App.Config.WithPolicy.Enable {
					err := app.App.DBService.AddServerLoginRecord(&db.AddSshLoginRequest{
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

func getServerApproveMenu(server config.Server) func(int, *MenuItem, *ssh.Session, []*MenuItem) []*MenuItem {
	return func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) []*MenuItem {
		sshd.ErrorInfo(fmt.Errorf("No permission for %s,Please apply for permission", server.Name), sess)
		var menu []*MenuItem
		menu = append(menu, &MenuItem{
			Label: fmt.Sprintf("Only this server: %s", server.Host),
			Info: map[string]string{
				serverInfoKey: server.Name,
			},
			SubMenuTitle: SelectAction,
			GetSubMenu: getActionMenu(db.ServerFilter{
				IpAddr: tea.String(server.Host),
			}),
		})
		serverTeam := server.Tags.GetTeam()
		if serverTeam != nil {
			// 申请机器所在组权限
			menu = append(menu, &MenuItem{
				Label:        fmt.Sprintf("All Server with tag: Team=%s", *serverTeam),
				Info:         map[string]string{},
				SubMenuTitle: SelectAction,
				GetSubMenu: getActionMenu(db.ServerFilter{
					Team: serverTeam,
				}),
			})
		}
		return menu
	}
}

func getActionMenu(serverFilter db.ServerFilter) func(int, *MenuItem, *ssh.Session, []*MenuItem) []*MenuItem {
	return func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) []*MenuItem {
		sshd.Info("申请行为，包括连接，上传和下载文件的权限。", sess)
		var menu []*MenuItem
		for key, value := range db.DefaultPolicies {
			menu = append(menu, &MenuItem{
				Label:      fmt.Sprintf("申请 %s 权限", key),
				GetSubMenu: getExpireMenu(serverFilter, value),
			})
		}
		return menu
	}
}

func getExpireMenu(serverFilter db.ServerFilter, actions db.ArrayString) func(int, *MenuItem, *ssh.Session, []*MenuItem) []*MenuItem {
	return func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) []*MenuItem {
		var menu []*MenuItem
		sshd.Info("选择策略生效周期，到期后会自动删除策略。", sess)
		for expiredKey, value := range db.ExpireTimes {
			menu = append(menu, &MenuItem{
				Label:        fmt.Sprintf("策略有效期 %s", expiredKey),
				SubMenuTitle: "Summary",
				GetSubMenu:   getSureApplyMenu(serverFilter, actions, value),
			})
		}
		return menu
	}
}

func getSureApplyMenu(serverFilter db.ServerFilter, actions db.ArrayString, expiredDuration time.Duration) func(int, *MenuItem, *ssh.Session, []*MenuItem) []*MenuItem {
	return func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) []*MenuItem {
		expired := time.Now().Add(expiredDuration)
		policyNew := &db.PolicyMut{
			Actions:      actions,
			Users:        db.ArrayString{tea.String((*sess).User())},
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
				if app.App.Config.WithDingtalk.Enable {
					id, err := dingtalk.CreateApproval((*sess).User(), []dt.FormComponentValue{
						{
							Name:  tea.String("Teams"),
							Value: tea.String(tea.Prettify(policyNew.Groups)),
						},
						{
							Name:  tea.String("Assets"),
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
				policyId, err := app.App.DBService.CreatePolicy(policyNew, &approval_id)
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
