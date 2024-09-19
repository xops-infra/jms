package model

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/robfig/cron"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
)

var Conf *Config

func init() {
	Conf = &Config{}
}

// Config config
type Config struct {
	Profiles     []CreateProfileRequest `mapstructure:"profiles"` // 云账号配置，用来自动同步云服务器信息
	Proxys       []CreateProxyRequest   `mapstructure:"proxies"`  // ssh代理
	Keys         Keys                   `mapstructure:"keys"`
	LocalServers []LocalServer          `mapstructure:"localServers"` // 支持人工加入的服务器
	WithVideo    WithVideo              `mapstructure:"withVideo"`    // 视频存储
	WithLdap     WithLdap               `mapstructure:"withLdap"`     // 配置ldap
	WithSSHCheck WithSSHCheck           `mapstructure:"withSSHCheck"` // 配置服务器SSH可连接性告警
	WithDB       WithPolicy             `mapstructure:"withDB"`       // 需要进行权限管理则启用该配置，启用后会使用数据库进行权限管理
	WithDingtalk WithDingtalk           `mapstructure:"withDingtalk"` // 配置钉钉审批流程
}

type Keys []AddKeyRequest

// ToMapWithID convert to map with keyID
func (k Keys) ToMapWithID() map[string]AddKeyRequest {
	m := make(map[string]AddKeyRequest)
	for _, key := range k {
		m[*key.KeyID] = key
	}
	return m
}

func (k Keys) ToMapWithName() map[string]AddKeyRequest {
	m := make(map[string]AddKeyRequest)
	for _, key := range k {
		// log.Debugf("key: %v", tea.Prettify(key))
		m[*key.IdentityFile] = key
	}
	return m
}

// ToMap convert to map with privateIp
func ServerListToMap(s []Server) map[string]Server {
	m := make(map[string]Server)
	for _, server := range s {
		m[server.Host] = server
	}
	return m
}

type WithVideo struct {
	Enable   bool   `mapstructure:"enable"`
	Cron     string `mapstructure:"cron"`     // 定时任务默认 "0 0 3 * * *" 表示每天凌晨 3 点触发
	Dir      string `mapstructure:"dir"`      // 日志目录,默认/opt/jms/audit/
	KeepDays int    `mapstructure:"keepDays"` // 保留天数,默认 3 个月
}

type WithDingtalk struct {
	Enable      bool   `mapstructure:"enable"`
	AppKey      string `mapstructure:"appKey"`
	AppSecret   string `mapstructure:"appSecret"`
	ProcessCode string `mapstructure:"processCode"` // 审批流程编码
}

type WithPolicy struct {
	Enable bool     `mapstructure:"enable"`
	DBFile string   `mapstructure:"dbFile"`
	PG     PGConfig `mapstructure:"pg"`
}

type PGConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

func (c *PGConfig) GetUrl() string {
	return fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Shanghai",
		c.Host,
		c.Username,
		c.Password,
		c.Database,
		cast.ToString(c.Port),
	)
}

type WithSSHCheck struct {
	Enable bool     `mapstructure:"enable"`
	Alert  SSHAlert `mapstructure:"alert"`
	IPS    []string `mapstructure:"ips"`
}

// 目前只支持钉钉机器人群告警
type SSHAlert struct {
	RobotToken string `mapstructure:"robotToken"`
}

type WithLdap struct {
	Enable           bool     `mapstructure:"enable"`
	BindUser         string   `mapstructure:"bindUser"`
	BindPassword     string   `mapstructure:"bindPassword"`
	Host             string   `mapstructure:"host"`
	Port             int      `mapstructure:"port"`
	BaseDN           string   `mapstructure:"baseDN"`
	UserSearchFilter string   `mapstructure:"userSearchFilter"`
	Attributes       []string `mapstructure:"attributes"`
}

// load config from file
func LoadYaml(configFile string) {
	homedir := os.Getenv("HOME")

	if strings.HasPrefix(configFile, "~") {
		configFile = strings.Replace(configFile, "~", homedir, 1)
	}
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		panic(err)
	}
	err := viper.Unmarshal(Conf)
	if err != nil {
		panic(err)
	}

	configCheck()

}

func configCheck() {
	// 校验 corn配置是否正确
	if Conf.WithVideo.Enable {
		if _, err := cron.Parse(Conf.WithVideo.Cron); err != nil {
			panic(fmt.Errorf("cron parse error: %s", err))
		}
		err := os.MkdirAll(Conf.WithVideo.Dir, 0755)
		if err != nil {
			panic(err)
		}
	}
}

// type User struct {
// 	Username   string `yaml:"username"`
// 	HashPasswd string `yaml:"hashPasswd"`
// 	Admin      bool   `yaml:"admin"`
// 	PublicKey  string `yaml:"publickey"`
// }

// Server server
type Server struct {
	ID       string
	Name     string
	Host     string // 默认取私有 IP 第一个
	Port     int
	KeyPairs []*string // key pair name
	// Proxy    *CreateProxyRequest
	Profile  string
	Region   string
	Tags     model.Tags
	Status   model.InstanceStatus
	SSHUsers []SSHUser
}

type LocalServer struct {
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

// SSHUser ssh user
type SSHUser struct {
	UserName  string
	KeyName   string // pem file name, 这里是支持本地读取内容的
	Base64Pem string // base64 pem
	Password  string
}
