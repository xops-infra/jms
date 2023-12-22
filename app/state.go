package app

import (
	"strings"

	"github.com/patrickmn/go-cache"
	"github.com/xops-infra/multi-cloud-sdk/pkg/io"
	server "github.com/xops-infra/multi-cloud-sdk/pkg/service"
	"github.com/xops-infra/noop/log"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/core/policy"
	"github.com/xops-infra/jms/utils"
)

func init() {
	config.Load(appDir)
}

var App *Application

type Application struct {
	Debug     bool
	SshDir    string
	DT        *utils.RobotClient
	Ldap      *utils.Ldap
	Config    *config.Config
	Server    *server.ServerService
	Cache     *cache.Cache
	UserCache *cache.Cache // 用户缓存,用于显示用户负载
	// DBIo          db.DbIo
	PolicyService *policy.PolicyService
}

const (
	appDir = "/opt/jms/"
)

// Manager,Agent,Worker need to be initialized
func NewApplication(debug bool, sshDir string) *Application {
	App = &Application{
		SshDir:    sshDir,
		Debug:     debug,
		Config:    config.Conf,
		Cache:     cache.New(cache.NoExpiration, cache.NoExpiration),
		UserCache: cache.New(cache.NoExpiration, cache.NoExpiration),
	}

	if len(App.Config.Profiles) == 0 {
		panic("no profile found")
	}

	cloudIo := io.NewCloudClient(App.Config.Profiles)
	serverTencent := io.NewTencentClient(cloudIo)
	serverAws := io.NewAwsClient(cloudIo)
	App.Ldap = utils.NewLdap(App.Config.Ldap)
	App.Server = server.NewServer(App.Config.Profiles, serverAws, serverTencent)

	return App
}

func (app *Application) WithDingTalk() *Application {
	dt := utils.NewRobotClient()
	app.DT = dt
	return app
}

// 启用 Policy 规则的情况下，使用数据库记录规则信息
func (app *Application) WithPolicy() *Application {
	dbFile := config.Conf.WithPolicy.DBFile
	if !strings.HasSuffix(dbFile, ".db") {
		panic("db file must be end with .db")
	}
	if dbFile == "" {
		dbFile = "jms.db"
	}
	rdb, err := gorm.Open(
		sqlite.Open(appDir+dbFile),
		&gorm.Config{},
	)
	log.Infof("sqlite file: %s", appDir+dbFile)
	if err != nil {
		panic("无法连接到数据库")
	}
	// 初始化数据库
	rdb.AutoMigrate(
		&policy.Policy{}, &policy.User{}, &policy.Group{},
	)
	App.PolicyService = policy.NewPolicyService(rdb)
	return app
}
