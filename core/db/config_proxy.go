package db

import (
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/noop/log"
)

func (d *DBService) ListProxy() ([]config.CreateProxyRequest, error) {
	var proxies []config.Proxy
	err := d.DB.Where("is_delete is false").Find(&proxies).Order("created_at").Error
	if err != nil {
		return nil, err
	}
	var res []config.CreateProxyRequest
	for _, proxy := range proxies {
		res = append(res, config.CreateProxyRequest{
			Name:         tea.String(proxy.Name),
			Host:         tea.String(proxy.Host),
			Port:         tea.Int(proxy.Port),
			IPPrefix:     tea.String(proxy.IPPrefix),
			LoginUser:    tea.String(proxy.LoginUser),
			LoginPasswd:  tea.String(proxy.LoginPasswd),
			IdentityFile: tea.String(proxy.IdentityFile),
			KeyID:        tea.String(proxy.KeyID),
		})
	}
	return res, err
}

func (d *DBService) GetProxyByIP(ip string) (*config.CreateProxyRequest, error) {
	var proxies []config.Proxy
	err := d.DB.Where("is_delete is false").Find(&proxies).Error
	if err != nil {
		return nil, err
	}
	for _, proxy := range proxies {
		if strings.HasPrefix(ip, proxy.IPPrefix) {
			return &config.CreateProxyRequest{
				Name:         tea.String(proxy.Name),
				Host:         tea.String(proxy.Host),
				Port:         tea.Int(proxy.Port),
				IPPrefix:     tea.String(proxy.IPPrefix),
				LoginUser:    tea.String(proxy.LoginUser),
				LoginPasswd:  tea.String(proxy.LoginPasswd),
				IdentityFile: tea.String(proxy.IdentityFile),
				KeyID:        tea.String(proxy.KeyID),
			}, nil
		}
	}
	return nil, nil
}

func (d *DBService) CreateProxy(req config.CreateProxyRequest) (config.Proxy, error) {
	// 跳过已经存在的
	var count int64
	d.DB.Model(&config.Proxy{}).Where("name = ? and is_delete is false", req.Name).Count(&count)
	if count > 0 {
		return config.Proxy{}, fmt.Errorf("proxy name %s already exists", *req.Name)
	}
	proxy, err := req.ToProxy()
	if err != nil {
		return config.Proxy{}, err
	}
	proxy.UUID = uuid.New().String()
	log.Debugf(tea.Prettify(proxy))

	err = d.DB.Create(&proxy).Error
	return proxy, err
}

func (d *DBService) UpdateProxy(uuid string, req config.CreateProxyRequest) (config.Proxy, error) {
	err := d.DB.Where("uuid = ? and is_delete is false", uuid).Updates(req).Error
	if err != nil {
		return config.Proxy{}, err
	}
	var proxy config.Proxy
	err = d.DB.Where("uuid = ? and is_delete is false", uuid).First(&proxy).Error
	return proxy, err
}

func (d *DBService) DeleteProxy(uuid string) error {
	// 先找
	var proxy config.Proxy
	err := d.DB.Where("uuid = ? and is_delete is false", uuid).First(&proxy).Error
	if err != nil {
		return err
	}
	proxy.IsDelete = true
	return d.DB.Save(&proxy).Error
}
