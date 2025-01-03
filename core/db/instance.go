package db

import (
	"fmt"

	"github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
)

// 数据库加载服务器
func (d *DBService) LoadServer() (model.Servers, error) {

	var servers model.Servers
	err := d.DB.Find(&servers).Order("host ASC").Error
	if err != nil {
		return nil, err
	}
	servers.SortByName()
	return servers, nil
}

func (d *DBService) GetInstanceByHost(host string) (*model.Server, error) {
	var server model.Server
	err := d.DB.Where("host = ?", host).First(&server).Error
	if err != nil {
		return nil, err
	}
	return &server, nil
}

// 更新数据库服务器列表，支持删除没有的服务器
// 注意支持 passwd 字段可以保留
func (d *DBService) UpdateServerWithDelete(newServers []model.Server) error {
	// Step 1: Load existing servers
	var existingServers []model.Server
	err := d.DB.Find(&existingServers).Error
	if err != nil {
		return err
	}

	var manualPasswdServers model.Servers
	// find passwd not '' to reset new server
	err = d.DB.Where("passwd != ''").Find(&manualPasswdServers).Error
	if err != nil {
		return err
	}
	manualPasswdServersMap := manualPasswdServers.ToMap()

	// Step 2: Build a map of new servers for quick lookup
	newServerMap := make(map[string]model.Server)
	for id, server := range newServers {
		// find manual passwd move to new server
		if manual_server, found := manualPasswdServersMap[server.Host]; found {
			server.Passwd = manual_server.Passwd
			server.User = manual_server.User
			newServers[id] = server
			log.Infof("reset server from manual passwd: %s", server.Host)
		}
		newServerMap[server.Host] = server
	}

	// Step 3: Find servers to delete
	for _, existingServer := range existingServers {
		if _, found := newServerMap[existingServer.Host]; !found {
			// keep manual passwd server
			if existingServer.Passwd != "" {
				// 更新机器 Name 增加 [Offline]$Name 前缀方式 重新加回去
				existingServer.Name = fmt.Sprintf("[Offline]%s", existingServer.Name)
				newServers = append(newServers, existingServer)
				log.Infof("add offline server passwd: %s", existingServer.Host)
				continue
			}
			// Existing server not in the new list, delete it
			if err := d.DB.Delete(&existingServer).Error; err != nil {
				return err
			}
			log.Infof("delete server: %s", existingServer.Host)
		}
	}

	// Step 4: Save (insert or update) the new servers
	return d.DB.Save(&newServers).Error
}
