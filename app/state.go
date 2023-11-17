package app

import (
	"github.com/patrickmn/go-cache"
	dt "github.com/patsnapops/go-dingtalk-sdk-wrapper"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/utils"
	"github.com/xops-infra/multi-cloud-sdk/pkg/io"
	server "github.com/xops-infra/multi-cloud-sdk/pkg/service"
	"gorm.io/gorm"
)

func init() {
	// err := godotenv.Load(".env")
	// if err != nil {
	// 	panic(err)
	// }
}

var App *Application

type Application struct {
	Debug  bool
	SshDir string
	DT     *dt.DingTalkClient
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
		Config: config.Load(appDir),
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
	dt, err := dt.NewDingTalkClient(&dt.DingTalkConfig{
		AppKey:    app.Config.DingTalk.AppKey,
		AppSecret: app.Config.DingTalk.AppSecret,
	})
	if err != nil {
		panic(err)
	}
	dt.WithMiniProgramClient(app.Config.DingTalk.AgentId)
	app.DT = dt
	return app
}
