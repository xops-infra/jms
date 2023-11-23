package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/patsnapops/noop/log"
	"github.com/spf13/viper"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
)

// Config config
type Config struct {
	Servers  *map[string]Server
	Policies []Policy              `mapstructure:"policies"`
	Groups   []Group               `mapstructure:"groups"`
	Profiles []model.ProfileConfig `mapstructure:"profiles"`
	Ldap     Ldap                  `mapstructure:"ldap"`
	Proxies  []Proxy               `mapstructure:"proxies"`
	Keys     map[string]string     `mapstructure:"keys"`
	DingTalk DingTalk              `mapstructure:"dingtalk"`
	Monitor  Monitor               `mapstructure:"monitor"`
}

type DingTalk struct {
	RobotToken string `mapstructure:"robotToken"`
}

type Monitor struct {
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
func loadConfig(configFile string) error {
	homedir := os.Getenv("HOME")

	if strings.HasPrefix(configFile, "~") {
		configFile = strings.Replace(configFile, "~", homedir, 1)
	}
	fmt.Printf("config file: %s\n", configFile)
	viper.SetConfigFile(configFile)
	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("read config file error: %s", err)
	}

	// 使用fsnotify监视配置文件变化
	viper.WatchConfig()
	viper.OnConfigChange(func(e fsnotify.Event) {
		fmt.Println("Config file changed:", e.Name)
		err := viper.ReadInConfig()
		if err != nil {
			fmt.Println("Error reloading config:", err)
		} else {
			fmt.Println("Config reloaded successfully")
		}
	})

	return nil
}

func Load(configDir string) *Config {
	if !strings.HasSuffix(configDir, "/") {
		configDir = configDir + "/"
	}
	proxy := &Config{
		Servers: &map[string]Server{},
	}
	err := loadConfig(configDir + ".jms.yml")
	if err != nil {
		log.Fatalf(err.Error())
	}
	viper.Unmarshal(proxy)
	// fmt.Printf("%+v\n", proxy)
	return proxy
}

type User struct {
	Username   string `yaml:"username"`
	HashPasswd string `yaml:"hashPasswd"`
	Admin      bool   `yaml:"admin"`
	PublicKey  string `yaml:"publickey"`
}

// Server server
type Server struct {
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
