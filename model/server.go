package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
)

// Server server
type Server struct {
	ID       string               `gorm:"primaryKey;column:id;not null"`
	Name     string               `gorm:"column:name"`
	Host     string               `gorm:"column:host"` // 默认取私有 IP 第一个
	Port     int                  `gorm:"column:port"`
	KeyPairs StringSlice          `gorm:"column:key_pairs;type:json"` // key pair name
	User     string               `gorm:"column:user;default:''"`     // 用 KEY 的这里可以不写，在 key里面指定用户，如果带上 Passwd的 User必须有
	Passwd   string               `gorm:"column:passwd;default:''"`
	Profile  string               `gorm:"column:profile"`
	Region   string               `gorm:"column:region"`
	Tags     model.Tags           `gorm:"column:tags;type:json"`
	Status   model.InstanceStatus `gorm:"column:status"`
}

func (s *Server) TableName() string {
	return "server"
}

// 支持本地配置服务器
type ServerManual struct {
	Name   string `mapstructure:"name"`
	Host   string `mapstructure:"host"`
	Port   int    `mapstructure:"port"`
	User   string `mapstructure:"user"`
	Passwd string `mapstructure:"passwd"`
	// IdentityFile string `mapstructure:"identity_file"`
}

type Servers []Server

// 按名称排序
func (s Servers) SortByName() {
	sort.Slice(s, func(i, j int) bool {
		// log.Debugf("%s %s", s[i].Name, s[j].Name)
		return s[i].Name < s[j].Name
	})
}

// toMap
func (s Servers) ToMap() map[string]Server {
	res := make(map[string]Server)
	for _, server := range s {
		res[server.Host] = server
	}
	return res
}

type ServerFilter struct {
	Name    *string `json:"name"`     // 名字完全匹配，支持*
	IpAddr  *string `json:"ip_addr"`  // IP 地址完全匹配，支持* 匹配所有
	EnvType *string `json:"env_type"` // 机器 Tags 中的 EnvType，支持* 匹配所有
	Team    *string `json:"team"`     // 机器 Tags 中的 Team，支持* 匹配所有
}

func (a ServerFilter) ToV1() *ServerFilterV1 {
	res := &ServerFilterV1{}
	if a.Name != nil {
		res.Name = []string{*a.Name}
	}
	if a.IpAddr != nil {
		res.IpAddr = []string{*a.IpAddr}
	}
	if a.EnvType != nil {
		res.EnvType = []string{*a.EnvType}
	}
	if a.Team != nil {
		res.Team = []string{*a.Team}
	}
	return res
}

func (a ServerFilter) ToString() string {
	return fmt.Sprintf("%v", a)
}

func (a ServerFilter) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func (a *ServerFilter) Scan(value interface{}) error {
	bytesValue, _ := value.([]byte)
	return json.Unmarshal(bytesValue, a)
}
