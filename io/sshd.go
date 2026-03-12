package io

import (
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
)

type SshdIO struct {
	db           *db.DBService
	localServers map[string]model.ServerManual
}

func NewSshd(db *db.DBService, localServers map[string]model.ServerManual) *SshdIO {
	return &SshdIO{
		localServers: localServers,
		db:           db,
	}
}

// 依据 keyid 获取 sshuser 认证信息 支持同一个 KEY 配置多个登录用户的情况
func (i *SshdIO) GetSSHUserByKeyID(keyID string, keys []model.AddKeyRequest) ([]model.SSHUser, error) {
	var sshUsers []model.SSHUser
	for _, key := range keys {
		log.Debugf("keyid: %s key: %s", keyID, summarizeKey(key))
		if key.KeyID == nil || key.UserName == nil {
			continue
		}
		if *key.KeyID == keyID {
			sshUsers = append(sshUsers, model.SSHUser{
				KeyName:   *key.IdentityFile,
				UserName:  *key.UserName,
				Base64Pem: tea.StringValue(key.PemBase64),
			})
		}
	}
	log.Debugf("sshUsers: %s", summarizeSSHUsers(sshUsers))
	if len(sshUsers) == 0 {
		return nil, fmt.Errorf("key %s not found in jms", keyID)
	}
	return sshUsers, nil
}

// 依据 host获取服务器所有的 sshuser
// 支持在云上 key，还支持本地配置的 sshuser 通过 IP 匹配；
func (i *SshdIO) GetSSHUsersByHost(host string, servers map[string]model.Server, keys []model.AddKeyRequest) ([]model.SSHUser, error) {
	log.Debugf("GetSSHUsersByHost: %s servers: %d keys: %d", host, len(servers), len(keys))
	if len(servers) == 0 {
		return nil, fmt.Errorf("servers is empty")
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("keys is empty")
	}
	var newSshUsers []model.SSHUser
	if server, ok := servers[host]; ok {
		// 先组装带 passwd的 sshuser
		if server.Passwd != "" {
			if server.User == "" {
				log.Errorf("server %s user is empty, set to root. if not ok, please set user for server %s", host, host)
				// set user to root
				server.User = "root"
			}
			newSshUsers = append(newSshUsers, model.SSHUser{
				KeyName:  "manual_passwd",
				UserName: server.User,
				Password: server.Passwd,
			})
		}
		log.Debugf("GetSSHUsersByHost: %s key: %s", host, tea.Prettify(server.KeyPairs))
		for _, keyID := range server.KeyPairs {
			sshUsers, err := i.GetSSHUserByKeyID(keyID, keys)
			if err != nil {
				continue
			}
			newSshUsers = append(newSshUsers, sshUsers...)
		}
	} else {
		log.Errorf("server %s not found in jms", host)
	}

	// 再去本地配置（不需要了。现在直接在数据库组装 passwd）
	// if server, ok := i.localServers[host]; ok {
	// 	newSshUsers = append(newSshUsers, model.SSHUser{
	// 		KeyName:  "local_config",
	// 		UserName: server.User,
	// 		Password: server.Passwd,
	// 	})
	// }
	log.Debugf("newSshUsers for host: %s is %s", host, summarizeSSHUsers(newSshUsers))
	return newSshUsers, nil
}

// GetSSHUsersByHostLive loads latest server and key data from DB for the given host.
func (i *SshdIO) GetSSHUsersByHostLive(host string) (*model.Server, []model.SSHUser, error) {
	if i.db == nil {
		return nil, nil, fmt.Errorf("db not initialized")
	}
	server, err := i.db.GetInstanceByHost(host)
	if err != nil {
		return nil, nil, fmt.Errorf("get server by host %s error: %s", host, err.Error())
	}
	keys, err := i.db.InternalLoadKey()
	if err != nil {
		return server, nil, fmt.Errorf("load key error: %s", err.Error())
	}
	serversMap := map[string]model.Server{
		server.Host: *server,
	}
	sshUsers, err := i.GetSSHUsersByHost(server.Host, serversMap, keys)
	if err != nil {
		return server, nil, err
	}
	return server, sshUsers, nil
}

// 依据 scp的路径获取 sshuser和服务器
// 返回 sshuser 和 服务器 remotePath
func (i *SshdIO) GetSSHUserAndServerByScpPath(scpPath string) (*model.SSHUser, *model.Server, string, error) {

	args := strings.SplitN(scpPath, ":", 2)

	if len(args) < 2 {
		return nil, nil, "", fmt.Errorf("scp path %s invalid", scpPath)
	}

	inputServer, remotePath := args[0], args[1]
	selectorKeyName := ""
	if strings.Contains(inputServer, "#") {
		parts := strings.SplitN(inputServer, "#", 2)
		inputServer = parts[0]
		selector := parts[1]
		if selector == "" {
			return nil, nil, "", fmt.Errorf("scp path %s invalid: empty selector", scpPath)
		}
		if strings.HasPrefix(selector, "key_name=") {
			selectorKeyName = strings.TrimPrefix(selector, "key_name=")
			if selectorKeyName == "" {
				return nil, nil, "", fmt.Errorf("scp path %s invalid: key_name empty", scpPath)
			}
		} else {
			return nil, nil, "", fmt.Errorf("scp path %s invalid: unsupported selector %s", scpPath, selector)
		}
	}
	serverArgs := strings.SplitN(inputServer, "@", 2)
	if len(serverArgs) < 2 {
		return nil, nil, "", fmt.Errorf("scp path %s invalid", scpPath)
	}

	sshUsername, host := serverArgs[0], serverArgs[1]

	servers, err := i.db.LoadServer()
	if err != nil {
		return nil, nil, "", fmt.Errorf("load server error: %s", err.Error())
	}
	serversMap := servers.ToMap()

	keys, err := i.db.InternalLoadKey()
	if err != nil {
		return nil, nil, "", fmt.Errorf("load key error: %s", err.Error())
	}

	if server, ok := serversMap[host]; ok {
		if selectorKeyName != "" {
			for _, key := range keys {
				if key.IdentityFile == nil {
					continue
				}
				if *key.IdentityFile != selectorKeyName {
					continue
				}
				if key.UserName == nil || *key.UserName == "" {
					return nil, nil, "", fmt.Errorf("key_name %s has empty user_name in jms", selectorKeyName)
				}
				if *key.UserName != sshUsername {
					return nil, nil, "", fmt.Errorf("key_name %s belongs to user %s, not %s", selectorKeyName, *key.UserName, sshUsername)
				}
				return &model.SSHUser{
					KeyName:   *key.IdentityFile,
					UserName:  *key.UserName,
					Base64Pem: tea.StringValue(key.PemBase64),
				}, &server, remotePath, nil
			}
			return nil, nil, "", fmt.Errorf("key_name %s not found in jms", selectorKeyName)
		}
		// 获取机器ssh用户
		sshusers, err := i.GetSSHUsersByHost(host, serversMap, keys)
		if err != nil {
			return nil, nil, "", fmt.Errorf("get sshuser error: %s", err.Error())
		}
		for _, sshuser := range sshusers {
			if sshuser.UserName == sshUsername {
				return &sshuser, &server, remotePath, nil
			}
		}
		return nil, nil, "", fmt.Errorf("user %s not found in server %s, check server key pairs or set passwd in db", sshUsername, host)
	} else {
		return nil, nil, "", fmt.Errorf("server %s not found", host)
	}
}
