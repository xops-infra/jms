package app

import (
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/patrickmn/go-cache"
	dt "github.com/xops-infra/go-dingtalk-sdk-wrapper"
	"github.com/xops-infra/jms/core/db"
	model1 "github.com/xops-infra/jms/model"
	"github.com/xops-infra/jms/utils"
	"github.com/xops-infra/multi-cloud-sdk/pkg/io"
	"github.com/xops-infra/multi-cloud-sdk/pkg/model"
	server "github.com/xops-infra/multi-cloud-sdk/pkg/service"
	"github.com/xops-infra/noop/log"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var App *Application

type Application struct {
	Debug           bool
	HomeDir, SSHDir string // /opt/jms/
	Version         string
	RobotClient     *dt.RobotClient    // 钉钉机器人
	DingTalkClient  *dt.DingTalkClient // 钉钉APP使用审批流
	Ldap            *utils.Ldap
	Config          *model1.Config // 支持数据库和配置文件两种方式载入配置
	Cache           *cache.Cache

	JmsDBService *db.DBService
	McsServer    model.CommonContract
}

// Manager,Agent,Worker need to be initialized
// logdir 如果为空,默认为/opt/jms/logs
func NewApp(debug bool, logDir string, version string) *Application {
	if version == "" {
		version = "unknown"
	}

	App = &Application{
		HomeDir: "/opt/jms/",
		SSHDir:  "/opt/jms/.ssh/",
		Version: version,
		Debug:   debug,
		Config:  model1.Conf,
		Cache:   cache.New(cache.NoExpiration, cache.NoExpiration),
	}
	// init log
	if logDir == "" {
		logDir = App.HomeDir + "logs/"
	}
	logfile := strings.TrimSuffix(logDir, "/") + "/sshd.log"
	if debug {
		log.Default().WithLevel(log.DebugLevel).WithHumanTime(time.Local).WithFilename(logfile).Init()
	} else {
		log.Default().WithLevel(log.InfoLevel).WithHumanTime(time.Local).WithFilename(logfile).Init()
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
	go http.ListenAndServe(":6060", nil)
	log.Infof("start pprof on :6060")
	return App
}

func NewApiApplication(sshd bool) *Application {
	App = &Application{
		Debug:  sshd,
		Config: model1.Conf,
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
	log.Infof("success load mcs")
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
func (app *Application) WithDB(migrate bool) *Application {
	// 优先匹配 pg
	var dialector gorm.Dialector
	if app.Config.WithDB.PG.Database != "" {
		log.Debugf("with policy pg database: %s", app.Config.WithDB.PG.GetUrl())
		dialector = postgres.Open(app.Config.WithDB.PG.GetUrl())
	} else {
		dbFile := model1.Conf.WithDB.DBFile
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
	// 初始化数据库,debug模式不初始化
	if migrate {
		log.Infof("auto migrate db!")
		err = rdb.AutoMigrate(
			&model1.Policy{}, &model1.User{}, &model1.AuthorizedKey{},
			&model1.Key{}, &model1.Profile{}, &model1.Proxy{}, // 配置
			&model1.SSHLoginRecord{}, &model1.ScpRecord{}, // 审计
			&model1.Broadcast{},
			&model1.ShellTask{}, &model1.ShellTaskRecord{}, // 定时任务功能
		)
	}

	if err != nil {
		panic(err)
	}
	App.JmsDBService = db.NewJmsDbService(rdb)
	return app
}

// 抽出来在初始化用以及定时热加载数据库
func (app *Application) LoadFromDB() {
	log.Debugf("load from db")
	profiles, err := App.JmsDBService.LoadProfile()
	if err != nil {
		panic(err)
	}
	App.Config.Profiles = profiles
	// 支持mcs的动态init，因为 profiles 是动态变化的

	resp, err := App.JmsDBService.InternalLoadKey()
	if err != nil {
		log.Panicf("load keys failed: %v", err)
	}
	App.Config.Keys = resp

	proxys, err := App.JmsDBService.ListProxy()
	if err != nil {
		log.Panicf("load proxy failed: %v", err)
	}
	App.Config.Proxys = proxys

}

func DBProfilesToMcsProfiles(profiles []model1.CreateProfileRequest) []model.ProfileConfig {
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
