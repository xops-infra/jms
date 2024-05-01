package app

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/patrickmn/go-cache"
	dt "github.com/xops-infra/go-dingtalk-sdk-wrapper"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/jms/utils"
	"github.com/xops-infra/multi-cloud-sdk/pkg/io"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
	server "github.com/xops-infra/multi-cloud-sdk/pkg/service"
	"github.com/xops-infra/noop/log"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/xops-infra/jms/config"
)

var App *Application

type Application struct {
	Debug           bool
	HomeDir, SSHDir string // /opt/jms/
	Version         string
	RobotClient     *dt.RobotClient    // 钉钉机器人
	DingTalkClient  *dt.DingTalkClient // 钉钉APP使用审批流
	Ldap            *utils.Ldap
	Config          *config.Config // 支持数据库和配置文件两种方式载入配置
	Cache           *cache.Cache

	DBService *db.DBService
	McsServer model.CommonContract
}

// Manager,Agent,Worker need to be initialized
func NewSshdApplication(debug bool, version string) *Application {
	App = &Application{
		HomeDir: "/opt/jms/",
		SSHDir:  "/opt/jms/.ssh/",
		Version: version,
		Debug:   debug,
		Config:  config.Conf,
		Cache:   cache.New(cache.NoExpiration, cache.NoExpiration),
	}
	// mkdir
	err := os.MkdirAll(App.SSHDir, 0755)
	if err != nil {
		panic(err)
	}
	// 判断文件hostAuthorizedKeys是否存在，不存在则创建
	hostAuthorizedKeys := App.SSHDir + "authorized_keys"
	if !utils.FileExited(hostAuthorizedKeys) {
		// 600权限
		os.Create(hostAuthorizedKeys)
		os.Chmod(hostAuthorizedKeys, 0600)
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
	if app.Config.WithDB.PG.Database != "" {
		log.Debugf("with policy pg database: %s", app.Config.WithDB.PG.GetUrl())
		dialector = postgres.Open(app.Config.WithDB.PG.GetUrl())
	} else {
		dbFile := config.Conf.WithDB.DBFile
		if !strings.HasSuffix(dbFile, ".db") {
			panic("db file must be end with .db")
		}
		if dbFile == "" {
			dbFile = "jms.db"
		}
		// show dbfile directory
		log.Infof("db file path: %s/%s", filepath.Dir(dbFile), dbFile)
		dialector = sqlite.Open(dbFile)
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
		&db.Policy{}, &db.User{}, &db.AuthorizedKey{},
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
