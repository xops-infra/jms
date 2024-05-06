package db

import (
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/google/uuid"
	"github.com/xops-infra/noop/log"
	"gorm.io/gorm"
)

type CreateProxyRequest struct {
	Name         *string `json:"name" binding:"required" mapstructure:"name"` // 代理名称 唯一
	Host         *string `json:"host" mapstructure:"host"`
	Port         *int    `json:"port" mapstructure:"port"`
	IPPrefix     *string `json:"ip_prefix" mapstructure:"ip_prefix"`         // 适配哪些机器 IP 前缀使用 Proxy, 例如 192.168.1
	LoginUser    *string `json:"login_user" mapstructure:"login_user"`       // key超级用户 root ec2-user
	LoginPasswd  *string `json:"login_passwd" mapstructure:"login_passwd"`   // 密码或者key必须有一个, 优先使用密码
	KeyID        *string `json:"key_id" mapstructure:"key_id"`               // KeyID和IdentityFile都是用pem来验证，KeyID是唯一的，IdentityFile在名称命名时候不同账号可能会同名。当出现IdentityFile不唯一的时候可以用 keyID, 优先使用KeyID
	IdentityFile *string `json:"identity_file" mapstructure:"identity_file"` // KeyID和IdentityFile都是用pem来验证，KeyID是唯一的，IdentityFile在名称命名时候不同账号可能会同名。当出现IdentityFile不唯一的时候可以用 keyID, 优先使用KeyID
}

func (req *CreateProxyRequest) ToProxy() (Proxy, error) {
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
	if req.IdentityFile != nil {
		proxy.IdentityFile = *req.IdentityFile
	}
	if req.KeyID != nil {
		proxy.KeyID = *req.KeyID
	}
	if proxy.LoginPasswd == "" && proxy.IdentityFile == "" && proxy.KeyID == "" {
		return proxy, fmt.Errorf("login_passwd or identity_file or key_id is required")
	}
	return proxy, nil
}

type Proxy struct {
	gorm.Model   `json:"-"`
	IsDelete     bool   `gorm:"column:is_delete;type:boolean;not null;default:false"`
	UUID         string `gorm:"column:uuid;type:varchar(36);unique_index;not null"`
	KeyID        string `gorm:"column:key_id;type:varchar(255);not null;default:''"`
	Name         string `gorm:"column:name;type:varchar(255);not null"`
	Host         string `gorm:"column:host;type:varchar(255);not null"`
	Port         int    `gorm:"column:port;type:integer;not null"`
	IPPrefix     string `gorm:"column:ip_prefix;type:varchar(255);not null"`
	LoginUser    string `gorm:"column:login_user;type:varchar(255);not null"`
	LoginPasswd  string `gorm:"column:login_passwd;type:varchar(255);not null"`
	IdentityFile string `gorm:"column:identity_file;type:varchar(255);not null"`
}

func (Proxy) TableName() string {
	return "proxy"
}

func (d *DBService) ListProxy() ([]CreateProxyRequest, error) {
	var proxies []Proxy
	err := d.DB.Where("is_delete is false").Find(&proxies).Order("created_at").Error
	if err != nil {
		return nil, err
	}
	var res []CreateProxyRequest
	for _, proxy := range proxies {
		res = append(res, CreateProxyRequest{
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

func (d *DBService) GetProxyByIP(ip string) (*CreateProxyRequest, error) {
	var proxies []Proxy
	err := d.DB.Where("is_delete is false").Find(&proxies).Error
	if err != nil {
		return nil, err
	}
	for _, proxy := range proxies {
		if strings.HasPrefix(ip, proxy.IPPrefix) {
			return &CreateProxyRequest{
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

func (d *DBService) CreateProxy(req CreateProxyRequest) (Proxy, error) {
	// 跳过已经存在的
	var count int64
	d.DB.Model(&Proxy{}).Where("name = ? and is_delete is false", req.Name).Count(&count)
	if count > 0 {
		return Proxy{}, fmt.Errorf("proxy name %s already exists", *req.Name)
	}
	proxy, err := req.ToProxy()
	if err != nil {
		return Proxy{}, err
	}
	proxy.UUID = uuid.New().String()
	log.Debugf(tea.Prettify(proxy))

	err = d.DB.Create(&proxy).Error
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
