package io

import (
	"sync"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/core/db"
	. "github.com/xops-infra/jms/model"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
	"github.com/xops-infra/noop/log"
)

type InstanceIO struct {
	mcsS         model.CommonContract
	db           *db.DBService
	localServers []ServerManual
}

func NewInstance(m model.CommonContract, db *db.DBService, localServers []ServerManual) *InstanceIO {
	return &InstanceIO{
		mcsS: m,
		db:   db,

		localServers: localServers,
	}
}

func (i *InstanceIO) LoadServer() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorf("LoadServer panic: %v", err)
		}
	}()

	// 获取最新的 Profile 信息
	profiles, err := i.db.LoadProfile()
	if err != nil {
		log.Errorf("LoadProfile error: %v", err)
		return
	}

	mcsServers := make(map[string]model.Instance, 2000)
	startTime := time.Now()

	var wg sync.WaitGroup
	serverChan := make(chan model.Instance) // 建立一个通道用以安全传递实例

	// 启动一个 goroutine 处理实例数据
	go func() {
		for server := range serverChan {
			mcsServers[*server.InstanceID] = server
		}
	}()

	for _, profile := range profiles {
		if profile.Enabled {
			wg.Add(1)
			go func(profile CreateProfileRequest) {
				defer wg.Done()

				for _, region := range profile.Regions {
					input := model.InstanceFilter{}
					log.Debugf("get instances profile: %s region: %s", *profile.Name, region)

					for {
						resps, err := i.mcsS.DescribeInstances(*profile.Name, region, input)
						if err != nil {
							log.Errorf("%s %s DescribeInstances error: %v", *profile.Name, region, err)
							break
						}

						for _, instance := range resps.Instances {
							if instance.InstanceID == nil {
								log.Errorf("instance id is nil, %s", tea.Prettify(instance))
								continue
							}
							serverChan <- instance
						}

						if resps.NextMarker == nil {
							break
						}
						input.NextMarker = resps.NextMarker
					}
				}
				log.Infof("get instances profile: %s completed", *profile.Name)
			}(profile)
		}
	}

	wg.Wait()
	close(serverChan) // 关闭通道，避免 goroutine 泄露

	// 入库
	err = i.db.UpdateServerWithDelete(fmtServer(i.localServers, mcsServers))
	if err != nil {
		log.Errorf("update server error: %v", err)
		return
	}

	log.Infof("load server finished cost: %s, total: %d ", time.Since(startTime), len(mcsServers))
}

func fmtServer(localServers []ServerManual, instances map[string]model.Instance) Servers {
	var instanceAll Servers
	for _, instance := range instances {
		if instance.Status != model.InstanceStatusRunning {
			continue
		}

		// 支持一个机器多个 key
		var keyName []string
		if instance.KeyIDs == nil {
			log.Warnf("instance:%s key is nil", *instance.Name)
		} else {
			for _, key := range instance.KeyIDs {
				if key == nil {
					continue
				}
				keyName = append(keyName, *key)
			}
		}

		if len(instance.PrivateIP) < 1 {
			log.Errorf("instance: %s private ip is empty", *instance.Name)
			continue
		}
		newInstance := Server{
			ID:       *instance.InstanceID,
			Name:     tea.StringValue(instance.Name),
			Host:     *instance.PrivateIP[0],
			Port:     22,
			Profile:  instance.Profile,
			Region:   tea.StringValue(instance.Region),
			Status:   instance.Status,
			KeyPairs: keyName,
			Tags:     *instance.Tags,
		}
		instanceAll = append(instanceAll, newInstance)
	}

	// 载入自己配置服务器
	for _, server := range localServers {
		instanceAll = append(instanceAll, Server{
			ID:     server.Host,
			Name:   server.Name,
			Host:   server.Host,
			Port:   server.Port,
			Status: model.InstanceStatusRunning, // 配置加入的默认为running
		})
	}

	// instanceAll.SortByName()
	return instanceAll
}
