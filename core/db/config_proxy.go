package db

import (
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"github.com/xops-infra/noop/log"
	"gorm.io/gorm"
)

type CreateProxyRequest struct {
	Name        *string `json:"name" mapstructure:"name"`
	Host        *string `json:"host" mapstructure:"host"`
	Port        *int    `json:"port" mapstructure:"port"`
	IPPrefix    *string `json:"ip_prefix" mapstructure:"ip_prefix"`       // 适配哪些机器 IP 前缀使用 Proxy
	LoginUser   *string `json:"login_user" mapstructure:"login_user"`     // key超级用户 root ec2-user
	LoginPasswd *string `json:"login_passwd" mapstructure:"login_passwd"` // 密码或者key必须有一个, 优先使用密码
	LoginKeyID  *string `json:"login_key_id" mapstructure:"login_key_id"` // 密码或者key必须有一个
}

func (req *CreateProxyRequest) ToProxy() Proxy {
	var proxy Proxy
	if req.Name != nil {
		proxy.Name = *req.Name
	}
	if req.Host != nil {
		proxy.Host = *req.Host
	}
	if req.Port != nil {
		proxy.Port = *req.Port
	}
	if req.IPPrefix != nil {
		proxy.IPPrefix = *req.IPPrefix
	}
	if req.LoginUser != nil {
		proxy.LoginUser = *req.LoginUser
	}
	if req.LoginPasswd != nil {
		proxy.LoginPasswd = *req.LoginPasswd
	}
	if req.LoginKeyID != nil {
		proxy.LoginKey = *req.LoginKeyID
	}
	return proxy
}

type Proxy struct {
	gorm.Model  `json:"-"`
	IsDelete    bool   `gorm:"column:is_delete;type:boolean;not null;default:false"`
	UUID        string `gorm:"column:uuid;type:varchar(36);unique_index;not null"`
	Name        string `gorm:"column:name;type:varchar(255);not null"`
	Host        string `gorm:"column:host;type:varchar(255);not null"`
	Port        int    `gorm:"column:port;type:integer;not null"`
	IPPrefix    string `gorm:"column:ip_prefix;type:varchar(255);not null"`
	LoginUser   string `gorm:"column:login_user;type:varchar(255);not null"`
	LoginPasswd string `gorm:"column:login_passwd;type:varchar(255);not null"`
	LoginKey    string `gorm:"column:login_key;type:varchar(255);not null"`
}

func (Proxy) TableName() string {
	return "proxy"
}

func (d *DBService) ListProxy() ([]Proxy, error) {
	var proxies []Proxy
	err := d.DB.Where("is_delete is false").Find(&proxies).Order("created_at").Error
	return proxies, err
}

func (d *DBService) GetProxyByIP(ip string) (*CreateProxyRequest, error) {
	var proxies []Proxy
	err := d.DB.Where("is_delete is false").Find(&proxies).Error
	if err != nil {
		return nil, err
	}
	for _, proxy := range proxies {
		if strings.HasPrefix(ip, proxy.IPPrefix) {
			return &CreateProxyRequest{
				Name:        &proxy.Name,
				Host:        &proxy.Host,
				Port:        &proxy.Port,
				IPPrefix:    &proxy.IPPrefix,
				LoginUser:   &proxy.LoginUser,
				LoginPasswd: &proxy.LoginPasswd,
				LoginKeyID:  &proxy.LoginKey,
			}, nil
		}
	}
	return nil, nil
}

func (d *DBService) CreateProxy(req CreateProxyRequest) (Proxy, error) {
	proxy := req.ToProxy()
	proxy.UUID = uuid.New().String()
	log.Debugf(tea.Prettify(proxy))
	err := d.DB.Create(&proxy).Error
	return proxy, err
}

func (d *DBService) UpdateProxy(uuid string, req CreateProxyRequest) (Proxy, error) {
	err := d.DB.Where("uuid = ? and is_delete is false", uuid).Updates(req).Error
	if err != nil {
		return Proxy{}, err
	}
	var proxy Proxy
	err = d.DB.Where("uuid = ? and is_delete is false", uuid).First(&proxy).Error
	return proxy, err
}

func (d *DBService) DeleteProxy(uuid string) error {
	// 先找
	var proxy Proxy
	err := d.DB.Where("uuid = ? and is_delete is false", uuid).First(&proxy).Error
	if err != nil {
		return err
	}
	proxy.IsDelete = true
	return d.DB.Save(&proxy).Error
}
