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
	"github.com/xops-infra/jms/io"
	model1 "github.com/xops-infra/jms/model"
	"github.com/xops-infra/jms/utils"
	mcsIo "github.com/xops-infra/multi-cloud-sdk/pkg/io"
	server "github.com/xops-infra/multi-cloud-sdk/pkg/service"
	"github.com/xops-infra/noop/log"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var App *Application

type Core struct {
	DingTalkClient *dt.DingTalkClient // 钉钉APP使用审批流
	InstanceIO     *io.InstanceIO
}

type Sshd struct {
	PolicyIO    *io.PolicyIO
	KeyIO       *io.KeyIO
	SshdIO      *io.SshdIO
	RobotClient *dt.RobotClient // 钉钉机器人
	Ldap        *utils.Ldap
}

type Application struct {
	Debug           bool
	HomeDir, SSHDir string // /opt/jms/
	Version         string
	Config          *model1.Config // 支持数据库和配置文件两种方式载入配置
	Cache           *cache.Cache
	JmsDBService    *db.DBService
	Core            Core
	Sshd            Sshd
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
	// App.PolicyIO = io.NewPolicy(App.JmsDBService)
	// App.SshdIO = io.NewSshd(App.JmsDBService, App.Config.LocalServers.ToMapWithHost())
	// App.KeyIO = io.NewKey(App.JmsDBService)

	return App
}

// withLdap
func (app *Application) WithLdap() *Application {
	ldap, err := utils.NewLdap(App.Config.WithLdap)
	if err != nil {
		panic(err)
	}
	app.Sshd.Ldap = ldap
	return app
}

// withMcs sdk 查询云服务器
func (app *Application) WithMcs() *Application {

	profiles, err := app.JmsDBService.LoadProfile()
	if err != nil {
		log.Errorf("load profile error: %s", err)
		return app
	}
	_profiles := model1.DBProfilesToMcsProfiles(profiles)
	cloudIo := mcsIo.NewCloudClient(_profiles)
	serverTencent := mcsIo.NewTencentClient(cloudIo)
	serverAws := mcsIo.NewAwsClient(cloudIo)

	mcsServer := server.NewCommonService(_profiles, serverAws, serverTencent)
	App.Core.InstanceIO = io.NewInstance(mcsServer, app.JmsDBService, app.Config.LocalServers)
	log.Infof("success load mcs")
	return app
}

func (app *Application) WithRobot() *Application {
	app.Sshd.RobotClient = dt.NewRobotClient()
	return app
}

func (app *Application) WithDingTalk() *Application {
	client, _ := dt.NewDingTalkClient(&dt.DingTalkConfig{
		AppKey:    app.Config.WithDingtalk.AppKey,
		AppSecret: app.Config.WithDingtalk.AppSecret,
	})
	client.WithWorkflowClientV2().WithDepartClient().WithUserClient()
	app.Core.DingTalkClient = client
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
			&model1.Server{}, // 实例
		)
	}

	if err != nil {
		panic(err)
	}
	App.JmsDBService = db.NewJmsDbService(rdb)
	return app
}
