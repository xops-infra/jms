package db

import "github.com/xops-infra/jms/model"

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

// 更新数据库服务器列表，支持删除没有的服务器
func (d *DBService) UpdateServerWithDelete(newServers []model.Server) error {
	// Step 1: Load existing servers
	var existingServers []model.Server
	err := d.DB.Find(&existingServers).Error
	if err != nil {
		return err
	}

	// Step 2: Build a map of new servers for quick lookup
	newServerMap := make(map[string]model.Server)
	for _, server := range newServers {
		newServerMap[server.Host] = server
	}

	// Step 3: Find servers to delete
	for _, existingServer := range existingServers {
		if _, found := newServerMap[existingServer.Host]; !found {
			// Existing server not in the new list, delete it
			if err := d.DB.Delete(&existingServer).Error; err != nil {
				return err
			}
		}
	}

	// Step 4: Save (insert or update) the new servers
	return d.DB.Save(&newServers).Error
}
