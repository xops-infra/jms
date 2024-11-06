package instance

import (
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	. "github.com/xops-infra/jms/model"
)

// 支持数据库和配置文件两种方式获取 KEY以及 Profile.
func LoadServer(conf *Config) {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("LoadServer panic: %v", err)
		}
	}()

	var mcsServers []model.Instance
	startTime := time.Now()
	for _, profile := range conf.Profiles {
		log.Debugf(tea.Prettify(profile))
		log.Debugf("profile: %s is enabled: %t", *profile.Name, profile.Enabled)
		if !profile.Enabled {
			continue
		}
		for _, region := range profile.Regions {
			log.Debugf("get instances profile: %s region: %s", *profile.Name, region)
			input := model.InstanceFilter{}
			for {
				resps, err := app.App.McsServer.DescribeInstances(*profile.Name, region, input)
				if err != nil {
					log.Errorf("%s %s DescribeInstances error: %v", *profile.Name, region, err)
					break
				}
				mcsServers = append(mcsServers, resps.Instances...)
				if resps.NextMarker == nil {
					log.Warnf("break get instances profile: %s region: %s len: %d", *profile.Name, region, len(mcsServers))
					break
				}
				input.NextMarker = resps.NextMarker
				resps.Instances = nil
			}
		}
		// print every profile instances count
		log.Infof("get instances profile: %s len: %d", *profile.Name, len(mcsServers))
	}
	log.Debugf("conf.Keys.ToMapWithID(): %s", tea.Prettify(conf.Keys.ToMapWithID()))
	app.SetServers(fmtServer(mcsServers))
	log.Infof("load server finished cost: %s ", time.Since(startTime))
}

func fmtServer(instances []model.Instance) Servers {
	var instanceAll Servers
	for _, instance := range instances {
		if instance.Status != model.InstanceStatusRunning {
			continue
		}

		// 支持一个机器多个 key
		var keyName []*string
		if instance.KeyIDs == nil {
			log.Warnf("instance:%s key is nil", *instance.Name)
		} else {
			for _, key := range instance.KeyIDs {
				if key == nil {
					log.Debugf("instance: %s key is nil", *instance.Name)
					continue
				}
				// 解决 key大写不识别问题
				keyName = append(keyName, key)
			}
		}

		log.Debugf("instance: %s key: %s", tea.StringValue(instance.Name), tea.Prettify(keyName))

		if len(instance.PrivateIP) < 1 {
			log.Errorf("instance: %s private ip is empty", *instance.Name)
			continue
		}
		sshUser := fmtSuperUser(instance)
		newInstance := Server{
			ID:       *instance.InstanceID,
			Name:     tea.StringValue(instance.Name),
			Host:     *instance.PrivateIP[0],
			Port:     22,
			Profile:  instance.Profile,
			Region:   tea.StringValue(instance.Region),
			Status:   instance.Status,
			KeyPairs: keyName,
			SSHUsers: sshUser,
			Tags:     *instance.Tags,
		}
		instanceAll = append(instanceAll, newInstance)
	}
	// 载入自己配置服务器
	for _, server := range app.App.Config.LocalServers {
		instanceAll = append(instanceAll, Server{
			ID:     "local_config",
			Name:   server.Name,
			Host:   server.Host,
			Port:   server.Port,
			Status: model.InstanceStatusRunning, // 配置加入的默认为running
			SSHUsers: []SSHUser{
				{
					UserName: server.User,
					Password: server.Passwd,
				},
			},
		})
	}

	instanceAll.SortByName()
	return instanceAll
}

// 通过机器的密钥对 KeyIDs 获取对应的密钥Pem的路径
func getKeyPairByKeyIDS(keyIDS []*string) []AddKeyRequest {
	log.Debugf("keyIDS: %s", tea.Prettify(keyIDS))
	keysAll := make([]AddKeyRequest, 0)
	configKeys := app.App.Config.Keys.ToMapWithID()
	for _, keyID := range keyIDS {
		if keyID == nil {
			continue
		}
		if key, ok := configKeys[*keyID]; ok {
			keysAll = append(keysAll, key)
		}
	}
	return keysAll
}

// fmtSuperUser 支持多用户选择
func fmtSuperUser(instance model.Instance) []SSHUser {
	keys := getKeyPairByKeyIDS(instance.KeyIDs)
	log.Debugf("keys %s", tea.Prettify(keys))
	var sshUser []SSHUser
	for _, key := range keys {
		u := SSHUser{}
		if key.KeyID == nil {
			continue
		}
		// KeyName 是支持本地读取内容的
		if key.IdentityFile != nil {
			u.KeyName = tea.StringValue(key.IdentityFile)
		}
		// 支持密钥文件为 base64 的字符串
		if key.PemBase64 != nil {
			u.Base64Pem = tea.StringValue(key.PemBase64)
		}

		if strings.Contains(*instance.Platform, "Ubuntu") {
			u.UserName = "ubuntu"
		} else if *instance.Platform == "Linux/UNIX" {
			u.UserName = "ec2-user"
		} else {
			u.UserName = "root"
		}
		sshUser = append(sshUser, u)
	}
	// log.Debugf("ssh user: %v", sshUser)
	return sshUser
}
