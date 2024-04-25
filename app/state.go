package app

import (
	"strings"

	"github.com/patrickmn/go-cache"
	dt "github.com/xops-infra/go-dingtalk-sdk-wrapper"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/multi-cloud-sdk/pkg/io"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
	server "github.com/xops-infra/multi-cloud-sdk/pkg/service"
	"github.com/xops-infra/noop/log"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/utils"
)

var (
	AppDir   = "/opt/jms/"
	AuditDir = "/opt/jms/audit/"
)

func init() {
	// set default
	if config.Conf.APPSet.Audit.Dir != "" {
		AuditDir = config.Conf.APPSet.Audit.Dir
	}
	if config.Conf.APPSet.HomeDir != "" {
		AppDir = config.Conf.APPSet.HomeDir
	}
}

var App *Application

type Application struct {
	Debug          bool
	SshDir         string
	RobotClient    *dt.RobotClient    // 钉钉机器人
	DingTalkClient *dt.DingTalkClient // 钉钉APP使用审批流
	Ldap           *utils.Ldap
	Config         *config.Config // 支持数据库和配置文件两种方式载入配置
	Cache          *cache.Cache

	DBService *db.DBService
	McsServer model.CommonContract
}

// Manager,Agent,Worker need to be initialized
func NewSshdApplication(debug bool, sshDir string) *Application {
	App = &Application{
		SshDir: sshDir,
		Debug:  debug,
		Config: config.Conf,
		Cache:  cache.New(cache.NoExpiration, cache.NoExpiration),
	}

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

// withMcs
func (app *Application) WithMcs() *Application {
	if len(App.Config.Profiles) == 0 {
		panic("请配置 profiles")
	}
	profiles := DBProfilesToMcsProfiles(app.Config.Profiles)
	cloudIo := io.NewCloudClient(profiles)
	serverTencent := io.NewTencentClient(cloudIo)
	serverAws := io.NewAwsClient(cloudIo)
	App.McsServer = server.NewCommonService(profiles, serverAws, serverTencent)
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
		log.Debugf("with policy pg database: %s", app.Config.WithPolicy.PG.GetUrl())
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
		gormConfig.Logger = logger.Default.LogMode(logger.Silent)
	}

	rdb, err := gorm.Open(dialector, gormConfig)
	if err != nil {
		panic("无法连接到数据库")
	}
	// 初始化数据库
	rdb.AutoMigrate(
		&db.Policy{}, &db.User{},
		&db.Key{}, &db.Profile{}, &db.Proxy{}, // 配置
		&db.SSHLoginRecord{}, &db.ScpRecord{}, // 审计
	)
	App.DBService = db.NewDbService(rdb)
	app.LoadFromDB()
	return app
}

// 抽出来在初始化用以及定时热加载数据库
func (app *Application) LoadFromDB() {
	log.Debugf("load from db")
	profiles, err := App.DBService.LoadProfile()
	if err != nil {
		panic(err)
	}
	App.Config.Profiles = profiles

	resp, err := App.DBService.InternalLoad()
	if err != nil {
		log.Panicf("load keys failed: %v", err)
	}
	App.Config.Keys = resp

	proxys, err := App.DBService.ListProxy()
	if err != nil {
		log.Panicf("load proxy failed: %v", err)
	}
	App.Config.Proxys = proxys
}

func DBProfilesToMcsProfiles(profiles []db.CreateProfileRequest) []model.ProfileConfig {
	var mcsProfiles []model.ProfileConfig
	for _, profile := range profiles {
		mcsProfiles = append(mcsProfiles, model.ProfileConfig{
			Name:  *profile.Name,
			AK:    *profile.AK,
			SK:    *profile.SK,
			Cloud: model.Cloud(*profile.Cloud),
		})
	}
	return mcsProfiles
}
