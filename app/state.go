package app

import (
	"strings"

	"github.com/patrickmn/go-cache"
	dt "github.com/xops-infra/go-dingtalk-sdk-wrapper"
	"github.com/xops-infra/multi-cloud-sdk/pkg/io"
	server "github.com/xops-infra/multi-cloud-sdk/pkg/service"
	"github.com/xops-infra/noop/log"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/core/policy"
	"github.com/xops-infra/jms/utils"
)

const (
	AppDir   = "/opt/jms/"
	AuditDir = "/opt/jms/audit/"
)

var App *Application

type Application struct {
	Debug          bool
	SshDir         string
	RobotClient    *dt.RobotClient    // 钉钉机器人
	DingTalkClient *dt.DingTalkClient // 钉钉APP使用审批流
	Ldap           *utils.Ldap
	Config         *config.Config
	Server         *server.ServerService
	Cache          *cache.Cache
	UserCache      *cache.Cache // 用户缓存,用于显示用户负载
	// DBIo          db.DbIo
	PolicyService *policy.PolicyService
}

// Manager,Agent,Worker need to be initialized
func NewSshdApplication(debug bool, sshDir string) *Application {
	App = &Application{
		SshDir:    sshDir,
		Debug:     debug,
		Config:    config.Conf,
		Cache:     cache.New(cache.NoExpiration, cache.NoExpiration),
		UserCache: cache.New(cache.NoExpiration, cache.NoExpiration),
	}

	if len(App.Config.Profiles) == 0 {
		panic("请配置 profiles")
	}

	cloudIo := io.NewCloudClient(App.Config.Profiles)
	serverTencent := io.NewTencentClient(cloudIo)
	serverAws := io.NewAwsClient(cloudIo)
	App.Server = server.NewServer(App.Config.Profiles, serverAws, serverTencent)

	return App
}

func NewApiApplication() *Application {
	App = &Application{
		Config: config.Conf,
	}

	return App
}

// withLdap
func (app *Application) WithLdap() *Application {
	ldap, err := utils.NewLdap(App.Config.WithLdap)
	if err != nil {
		panic(err)
	}
	app.Ldap = ldap
	return app
}

func (app *Application) WithRobot() *Application {
	app.RobotClient = dt.NewRobotClient()
	return app
}

func (app *Application) WithDingTalk() *Application {
	client, _ := dt.NewDingTalkClient(&dt.DingTalkConfig{
		AppKey:    app.Config.WithDingtalk.AppKey,
		AppSecret: app.Config.WithDingtalk.AppSecret,
	})
	client.WithWorkflowClientV2().WithDepartClient().WithUserClient()
	app.DingTalkClient = client
	return app
}

// 启用 Policy 规则的情况下，使用数据库记录规则信息
func (app *Application) WithPolicy() *Application {
	// 优先匹配 pg
	var dialector gorm.Dialector
	if app.Config.WithPolicy.PG.Database != "" {
		log.Infof("with policy pg database: %s", app.Config.WithPolicy.PG.Database)
		dialector = postgres.Open(app.Config.WithPolicy.PG.GetUrl())
	} else {
		dbFile := config.Conf.WithPolicy.DBFile
		if !strings.HasSuffix(dbFile, ".db") {
			panic("db file must be end with .db")
		}
		if dbFile == "" {
			dbFile = "jms.db"
		}
		dialector = sqlite.Open(AppDir + dbFile)
	}

	gormConfig := &gorm.Config{}
	if !app.Debug {
		log.Infof("set gorm logger to silent")
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	rdb, err := gorm.Open(dialector)
	if err != nil {
		panic("无法连接到数据库")
	}
	// 初始化数据库
	rdb.AutoMigrate(
		&policy.Policy{}, &policy.User{},
	)
	App.PolicyService = policy.NewPolicyService(rdb)
	return app
}
