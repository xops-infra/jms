package app

import (
	"github.com/patrickmn/go-cache"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/utils"
	"github.com/xops-infra/multi-cloud-sdk/pkg/io"
	server "github.com/xops-infra/multi-cloud-sdk/pkg/service"
	"gorm.io/gorm"
)

func init() {
	config.Load(appDir)
}

var App *Application

type Application struct {
	Debug  bool
	SshDir string
	DT     *utils.RobotClient
	Ldap   *utils.Ldap
	Config *config.Config
	Server *server.ServerService
	DB     *gorm.DB
	Cache  *cache.Cache
}

const (
	appDir = "/opt/jms/"
)

// Manager,Agent,Worker need to be initialized
func NewApplication(debug bool, sshDir string) *Application {
	App = &Application{
		SshDir: sshDir,
		Debug:  debug,
		DB:     utils.NewSQLite(),
		Config: config.Conf,
		Cache:  cache.New(cache.NoExpiration, cache.NoExpiration),
	}

	if len(App.Config.Profiles) == 0 {
		panic("no profile found")
	}

	cloudIo := io.NewCloudClient(App.Config.Profiles)
	serverTencent := io.NewTencentClient(cloudIo)
	serverAws := io.NewAwsClient(cloudIo)
	App.Ldap = utils.NewLdap(App.Config.Ldap)
	App.Server = server.NewServer(App.Config.Profiles, serverAws, serverTencent)

	// 初始化数据库
	if App.DB != nil {
		App.DB.AutoMigrate(
			&utils.Policy{},
		)
	}
	return App
}

func (app *Application) WithDingTalk() *Application {
	dt := utils.NewRobotClient()
	app.DT = dt
	return app
}
