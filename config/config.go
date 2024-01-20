package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/fsnotify/fsnotify"
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
	Profiles     []model.ProfileConfig `mapstructure:"profiles"`     // 云账号配置，用来自动同步云服务器信息
	Proxies      []Proxy               `mapstructure:"proxies"`      // ssh代理
	Keys         map[string]string     `mapstructure:"keys"`         // ssh key pair
	WithLdap     WithLdap              `mapstructure:"withLdap"`     // 配置ldap
	WithSSHCheck WithSSHCheck          `mapstructure:"withSSHCheck"` // 配置服务器SSH可连接性告警
	WithPolicy   WithPolicy            `mapstructure:"withPolicy"`   // 需要进行权限管理则启用该配置，启用后会使用数据库进行权限管理
	WithDingtalk WithDingtalk          `mapstructure:"withDingtalk"` // 配置钉钉审批流程
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
func Load(configFile string) {
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

	// 使用fsnotify监视配置文件变化
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		err := viper.ReadInConfig()
		if err != nil {
			fmt.Printf("config file changed error: %s\n", err)
		} else {
			Conf = &Config{}
			viper.Unmarshal(Conf)
			fmt.Println("config file changed:", e.Name, tea.Prettify(Conf))
		}
	})

}

type User struct {
	Username   string `yaml:"username"`
	HashPasswd string `yaml:"hashPasswd"`
	Admin      bool   `yaml:"admin"`
	PublicKey  string `yaml:"publickey"`
}

// Server server
type Server struct {
	ID       string
	Name     string
	Host     string
	Port     int
	KeyPair  *string // key pair name
	Proxy    *Proxy
	Profile  string
	Region   string
	Tags     model.Tags
	Status   model.InstanceStatus
	SSHUsers *map[string]*SSHUser
}

type Proxy struct {
	Name     string   `mapstructure:"name"`
	Host     string   `mapstructure:"host"`
	Port     int      `mapstructure:"port"`
	SSHUsers *SSHUser `mapstructure:"sshUsers"`
	IPPrefix string   `mapstructure:"ipPrefix"`
}

// SSHUser ssh user
type SSHUser struct {
	SSHUsername  string
	IdentityFile string
	Password     string
}
