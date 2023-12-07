package pui

import (
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/elfgzp/ssh"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/core/instance"
	"github.com/xops-infra/jms/core/sshd"
)

func GetServersMenuV2(user, timeout string) []*MenuItem {
	menu := make([]*MenuItem, 0)
	servers := instance.GetServers()
	for serverKey, server := range servers {
		if matchPolicy(user, server, *app.App.Config) {
			info := make(map[string]string, 0)
			info[serverInfoKey] = *server.KeyPair
			info[serverHost] = server.Host
			for _, sshUser := range *server.SSHUsers {
				info[serverUser] += fmt.Sprintf("%s: %s", sshUser.SSHUsername, sshUser.IdentityFile)
			}
			if server.Proxy != nil {
				info[serverProxy] = server.Proxy.Host
				info[serverProxyUser] = server.Proxy.SSHUsers.SSHUsername
				info[serverProxyKeyIdentityFile] = server.Proxy.SSHUsers.IdentityFile
			}
			log.Debugf("server info: %v", tea.Prettify(info))

			subMenu := &MenuItem{
				Label:        fmt.Sprintf("%s\t%s\t%s", server.ID, server.Host, server.Name),
				Info:         info,
				SubMenuTitle: fmt.Sprintf("%s '%s'", UserLoginLabel, server.Name),
				GetSubMenu:   GetServerSSHUsersMenu(server, timeout),
			}
			menu = append(menu, subMenu)
		} else {
			log.Debugf("skip %s %v for %s", serverKey, server, user)
		}
	}
	// sort menu
	menu = sortMenu(menu)
	return menu
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

// match policy
func matchPolicy(user string, server config.Server, conf config.Config) bool {
	// 查询用户所在组
	var group string
	for _, _group := range conf.Groups {
		for _, userInGroup := range _group.Users {
			if userInGroup == user {
				group = _group.Name
				break
			}
		}
	}
	switch group {
	case "admin":
		return true
	default:
		// 依据策略判断是否允许
		for _, policy := range conf.Policies {
			if !policy.Enabled {
				continue
			}
			for _, filterGroup := range policy.Groups {
				if filterGroup == "*" || filterGroup == group {
					// 继续判断服务器过滤条件
					return matchTag(policy.ServerFilter, server.Tags)
				}
			}
		}
	}
	return false
}

func matchTag(configTag, serverTag model.Tags) bool {
	for _, configTagItem := range configTag {
		if strings.HasPrefix(configTagItem.Value, "!") {
			for _, serverTagItem := range serverTag {
				if configTagItem.Key == serverTagItem.Key {
					if strings.TrimPrefix(configTagItem.Value, "!") == serverTagItem.Value {
						return false
					}
				}
			}
			return true
		} else {
			for _, serverTagItem := range serverTag {
				if configTagItem.Key == serverTagItem.Key {
					if configTagItem.Value == serverTagItem.Value {
						return true
					}
				}
			}
			return false
		}
	}
	return false
}

func GetServerSSHUsersMenu(server config.Server, timeout string) func(int, *MenuItem, *ssh.Session, []*MenuItem) []*MenuItem {
	return func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) []*MenuItem {
		var menu []*MenuItem
		for key, sshUser := range *server.SSHUsers {
			info := make(map[string]string, 0)
			info[serverInfoKey] = key
			sshUserMenu := &MenuItem{
				Label: fmt.Sprintf("%s: %s", *server.KeyPair, key),
				Info:  info,
			}
			sshUserMenu.SelectedFunc = func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) error {
				if server.Status != model.InstanceStatusRunning {
					return fmt.Errorf("%s status %s, can not login", server.Host, strings.ToLower(string(server.Status)))
				}
				err := sshd.NewTerminal(server, sshUser, sess, timeout)
				if err != nil {
					return err
				}
				return nil
			}
			menu = append(menu, sshUserMenu)
		}
		return menu
	}
}
