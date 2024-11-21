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
	localServers map[string]model.LocalServer
}

func NewSshd(db *db.DBService, localServers map[string]model.LocalServer) *SshdIO {
	return &SshdIO{
		localServers: localServers,
		db:           db,
	}
}

// 依据 keyid 获取 sshuser 认证信息 支持同一个 KEY 配置多个登录用户的情况
func (i *SshdIO) GetSSHUserByKeyID(keyID string, keys []model.AddKeyRequest) ([]model.SSHUser, error) {
	var sshUsers []model.SSHUser
	for _, key := range keys {
		log.Debugf("keyid: %s key: %s", keyID, tea.Prettify(key))
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
	log.Debugf("sshUsers: %s", tea.Prettify(sshUsers))
	if len(sshUsers) == 0 {
		return nil, fmt.Errorf("key %s not found in jms", keyID)
	}
	return sshUsers, nil
}

// 依据 host获取服务器所有的 sshuser
func (i *SshdIO) GetSSHUsersByHost(host string, servers map[string]model.Server, keys []model.AddKeyRequest) ([]model.SSHUser, error) {
	var newSshUsers []model.SSHUser
	if server, ok := servers[host]; ok {
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

	// 再去本地配置
	if server, ok := i.localServers[host]; ok {
		newSshUsers = append(newSshUsers, model.SSHUser{
			KeyName:  "local_config",
			UserName: server.User,
			Password: server.Passwd,
		})
	}
	log.Debugf("newSshUsers for host: %s is %s", host, tea.Prettify(newSshUsers))
	return newSshUsers, nil
}

// 依据 scp的路径获取 sshuser和服务器
// 返回 sshuser 和 服务器 remotePath
func (i *SshdIO) GetSSHUserAndServerByScpPath(scpPath string) (*model.SSHUser, *model.Server, string, error) {

	args := strings.SplitN(scpPath, ":", 2)

	if len(args) < 2 {
		return nil, nil, "", fmt.Errorf("scp path %s invalid", scpPath)
	}

	inputServer, remotePath := args[0], args[1]
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
		return nil, nil, "", fmt.Errorf("user %s not found in server %s", sshUsername, host)
	} else {
		return nil, nil, "", fmt.Errorf("server %s not found", host)
	}
}
