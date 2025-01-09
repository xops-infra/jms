package app

import (
	"fmt"
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
	"github.com/xops-infra/jms/model"
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

type Schedule struct {
	DingTalkClient *dt.DingTalkClient // 钉钉APP使用审批流
	RobotClient    *dt.RobotClient    // 钉钉机器人
	InstanceIO     *io.InstanceIO
}

type Sshd struct {
	SshdIO *io.SshdIO
	Ldap   *utils.Ldap
}

type Application struct {
	Debug           bool
	HomeDir, SSHDir string // /opt/jms/
	Version         string
	Config          *model.Config // 支持数据库和配置文件两种方式载入配置
	Cache           *cache.Cache

	DBIo *db.DBService

	Schedule Schedule
	Sshd     Sshd
}

// Manager,Agent,Worker need to be initialized
// logdir 如果为空,默认为/opt/jms/logs
func NewApplication(debug bool, logDir, version, config string) *Application {
	if version == "" {
		version = "unknown"
	}

	App = &Application{
		HomeDir: "/opt/jms/",
		SSHDir:  "/opt/jms/.ssh/",
		Version: version,
		Debug:   debug,
		Config:  model.InitConfig(config),
		Cache:   cache.New(cache.NoExpiration, cache.NoExpiration),
	}

	// init log
	if logDir == "" {
		logDir = App.HomeDir + "logs/"
	}

	err := os.MkdirAll(utils.FilePath(logDir), 0755)
	if err != nil {
		log.Panicf("create log dir failed: %s", err.Error())
	}

	logfile := strings.TrimSuffix(logDir, "/") + "/app.log"
	if debug {
		log.Default().WithLevel(log.DebugLevel).WithHumanTime(time.Local).WithFilename(logfile).Init()
	} else {
		log.Default().WithLevel(log.InfoLevel).WithHumanTime(time.Local).WithFilename(logfile).Init()
	}
	go http.ListenAndServe(":6060", nil)

	fmt.Println("jms version: ", App.Version)
	fmt.Println("log file: ", logfile)
	fmt.Println("ssh dir: ", App.SSHDir)
	fmt.Println("home dir: ", App.HomeDir)
	fmt.Println("pprof: localhost:6060")

	return App
}

// withMcs sdk 查询云服务器
func (app *Application) WithMcs() *Application {

	profiles, err := app.DBIo.LoadProfile()
	if err != nil {
		log.Errorf("load profile error: %s", err)
		return app
	}
	_profiles := model.DBProfilesToMcsProfiles(profiles)
	cloudIo := mcsIo.NewCloudClient(_profiles)
	serverTencent := mcsIo.NewTencentClient(cloudIo)
	serverAws := mcsIo.NewAwsClient(cloudIo)

	mcsServer := server.NewCommonService(_profiles, serverAws, serverTencent)
	App.Schedule.InstanceIO = io.NewInstance(mcsServer, app.DBIo, app.Config.LocalServers)
	log.Infof("success load mcs")
	return app
}

func (app *Application) WithRobot() *Application {
	return app
}

func (app *Application) WithDingTalk() *Application {
	client, _ := dt.NewDingTalkClient(&dt.DingTalkConfig{
		AppKey:    app.Config.WithDingtalk.AppKey,
		AppSecret: app.Config.WithDingtalk.AppSecret,
	})
	client.WithWorkflowClientV2().WithDepartClient().WithUserClient()
	app.Schedule.DingTalkClient = client
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
		dbFile := app.Config.WithDB.DBFile
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
			&model.Policy{}, &model.User{}, &model.AuthorizedKey{},
			&model.Key{}, &model.Profile{}, &model.Proxy{}, // 配置
			&model.SSHLoginRecord{}, &model.ScpRecord{}, // 审计
			&model.Broadcast{},
			&model.ShellTask{}, &model.ShellTaskRecord{}, // 定时任务功能
			&model.Server{}, // 实例
		)
	}

	if err != nil {
		panic(err)
	}
	App.DBIo = db.NewJmsDbService(rdb)
	return app
}
