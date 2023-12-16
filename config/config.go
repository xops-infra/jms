package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
)

var Conf *Config

func init() {
	Conf = &Config{}
}

// Config config
type Config struct {
	Profiles     []model.ProfileConfig `mapstructure:"profiles"`     // 支持配置动态加载
	Ldap         Ldap                  `mapstructure:"ldap"`         // 支持配置动态加载
	Proxies      []Proxy               `mapstructure:"proxies"`      // 支持配置动态加载
	Keys         map[string]string     `mapstructure:"keys"`         // 支持配置动态加载
	DingTalk     DingTalk              `mapstructure:"dingtalk"`     // 支持配置动态加载
	WithSSHCheck WithSSHCheck          `mapstructure:"withSSHCheck"` // 支持配置动态加载
	WithPolicy   WithPolicy            `mapstructure:"withPolicy"`
}

type WithPolicy struct {
	DBFile string `mapstructure:"dbFile"`
}

type DingTalk struct {
	RobotToken string `mapstructure:"robotToken"`
}

type WithSSHCheck struct {
	IPS []string `mapstructure:"ips"`
}

type Ldap struct {
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
	configFile = strings.TrimSuffix(configFile, "/") + "/.jms.yml"
	fmt.Printf("config file: %s\n", configFile)
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
