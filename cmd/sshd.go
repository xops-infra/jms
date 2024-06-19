package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/elfgzp/ssh"
	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron"
	"github.com/spf13/cobra"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/dingtalk"
	"github.com/xops-infra/jms/core/instance"
	"github.com/xops-infra/jms/core/jump"
	"github.com/xops-infra/jms/core/sshd"
	appConfig "github.com/xops-infra/jms/model"
	"github.com/xops-infra/jms/utils"
)

var (
	logDir   string
	timeOut  int // s
	sshdPort int
)

var sshdCmd = &cobra.Command{
	Use:   "sshd",
	Short: "start sshd server as proxy server",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		appConfig.LoadYaml(config)

		err := os.MkdirAll(utils.FilePath(logDir), 0755)
		if err != nil {
			log.Fatalf("create log dir failed: %s", err.Error())
		}

		// init app
		_app := app.NewSshdApplication(debug, logDir, rootCmd.Version)

		if app.App.Config.WithLdap.Enable {
			log.Infof("enable ldap")
			_app.WithLdap()
		}

		if app.App.Config.WithSSHCheck.Enable {
			log.Infof("enable dingtalk")
			_app.WithRobot()
		}

		if app.App.Config.WithDingtalk.Enable {
			log.Infof("enable dingtalk")
			_app.WithDingTalk()
		}

		if app.App.Config.WithDB.Enable {
			_app.WithDB(!debug)
			_app.LoadFromDB() // 加载数据库配置
			log.Infof("enable db")
		}

		if app.App.Config.WithDingtalk.Enable {
			if !app.App.Config.WithDB.Enable {
				app.App.Config.WithDingtalk.Enable = false
				log.Warnf("dingtalk enable but db not enable, disable dingtalk")
			} else {
				log.Infof("enable api dingtalk Approve")
			}
		}

		app.App.WithMcs()
		instance.LoadServer(app.App.Config)

		ssh.Handle(func(sess ssh.Session) {
			defer func() {
				if e, ok := recover().(error); ok {
					log.Error(e.Error())
				}
			}()
			sessionHandler(&sess)
		})

		var wrapped *wrappedConn
		hostKeyFile := app.App.SSHDir + "id_rsa"
		// log.Panicf(hostKeyFile)
		if !utils.FileExited(hostKeyFile) {
			sshd.GenKey(hostKeyFile)
		}

		log.Infof("starting ssh server on port %d timeout %d...", sshdPort, timeOut)

		if !debug {
			// 服务启动后再启动定时任务
			go startScheduler()
		}
		err = ssh.ListenAndServe(
			fmt.Sprintf(":%d", sshdPort),
			nil,
			ssh.PasswordAuth(passwordAuth),
			ssh.PublicKeyAuth(publicKeyAuth),
			ssh.HostKeyFile(utils.FilePath(hostKeyFile)),
			ssh.WrapConn(func(ctx ssh.Context, conn net.Conn) net.Conn {
				conn.SetDeadline(time.Now().Add(5 * time.Second))
				wrapped = &wrappedConn{conn, 0}
				return wrapped
			}),
		)
		if err != nil {
			log.Panic(err.Error())
		}
	},
}

type wrappedConn struct {
	net.Conn
	written int32
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.AddCommand(sshdCmd)
	sshdCmd.Flags().IntVar(&sshdPort, "port", 22222, "ssh port")
	sshdCmd.Flags().StringVar(&logDir, "log-dir", "/opt/jms/logs/", "log dir")
	sshdCmd.Flags().IntVar(&timeOut, "timeout", 1800, "ssh timeout")

}

func passwordAuth(ctx ssh.Context, pass string) bool {
	if app.App.Config.WithLdap.Enable {
		err := app.App.Ldap.Login(ctx.User(), pass)
		return err == nil
	}
	// 如果启用 policy策略，登录时需要验证用户密码
	if app.App.Config.WithDB.Enable {
		allow, err := app.App.JmsDBService.Login(ctx.User(), pass)
		if err != nil {
			log.Error(err.Error())
			return false
		}
		return allow
	} else {
		// 当 ladp和数据库都么启用的时候， 默认认证，jms/jms
		switch ctx.User() {
		case "jms":
			return pass == "jms"
		default:
			return false
		}
	}
}

// 支持authorized_keys读取 pub key 认证
// 还支持pubkey在数据库
func publicKeyAuth(ctx ssh.Context, key ssh.PublicKey) bool {
	if app.App.Config.WithDB.Enable {
		// 数据库读取数据认证
		return app.App.JmsDBService.AuthKey(ctx.User(), key)
	}
	// 否则走文件认证
	return utils.AuthFromFile(ctx, key, app.App.SSHDir)
}

func sessionHandler(sess *ssh.Session) {
	defer func() {
		(*sess).Close()
	}()
	user := (*sess).User()
	remote := (*sess).RemoteAddr()
	_, found := app.App.Cache.Get(user)
	if !found {
		app.App.Cache.Add(user, 1, cache.DefaultExpiration)
	}

	log.Infof("user: %s, remote addr: %s login success", user, remote)
	rawCmd := (*sess).RawCommand()
	log.Debugf("rawCmd: %s\n", rawCmd)
	cmd, args, err := sshd.ParseRawCommand(rawCmd)
	if err != nil {
		sshd.ErrorInfo(err, sess)
		return
	}
	log.Debugf("cmd: %s, args: %s\n", cmd, args)
	switch cmd {
	case "exec":
		execHandler(sess)
	case "scp":
		scpHandler(args, sess)
	case "exit":
		(*sess).Exit(0)
	case "ssh":
		sshHandler(sess)
	default:
		if strings.Contains(cmd, "umask") {
			// 版本问题导致的 cmd不一致问题
			execHandler(sess)
		}
		sshHandler(sess)
	}
}

func execHandler(sess *ssh.Session) {
	// 执行命令
	// 获取用户后续输入的 pubKey 存放到 authorized_keys 文件中
	pubKey, err := bufio.NewReader(*sess).ReadString('\n')
	if err != nil {
		log.Error(err.Error())
		return
	}
	if !strings.Contains(pubKey, "ssh-rsa") {
		sshd.ErrorInfo(errors.New("not ssh-rsa key"), sess)
		return
	}
	if app.App.Config.WithDB.Enable {
		// 数据库读取数据认证
		if err := app.App.JmsDBService.AddAuthorizedKey((*sess).User(), pubKey); err != nil {
			sshd.ErrorInfo(err, sess)
			log.Errorf("add authorized key error: %s", err.Error())
			return
		}
	} else {
		// 否则走文件认证
		err := utils.AddAuthToFile((*sess).User(), pubKey, app.App.SSHDir)
		if err != nil {
			sshd.ErrorInfo(err, sess)
			return
		}
	}
	// 退出
	(*sess).Exit(0)
}

func sshHandler(sess *ssh.Session) {
	jps := jump.NewSession(sess, time.Duration(timeOut)*time.Second)
	jps.Run()
}

func scpHandler(args []string, sess *ssh.Session) {
	err := sshd.ExecuteSCP(args, sess)
	if err != nil {
		sshd.ErrorInfo(err, sess)
		return
	}
}

// debug will not run
func startScheduler() {
	c := cron.New()
	time.Sleep(10 * time.Second) // 等待app初始化完成
	c.AddFunc("0 */2 * * * *", func() {
		instance.LoadServer(app.App.Config)
	})

	if app.App.Config.WithDB.Enable {
		log.Infof("enabled db config hot update, 2 min check once")
		// 启用定时热加载数据库配置,每 30s 检查一次
		c.AddFunc("*/30 * * * * *", func() {
			app.App.LoadFromDB()
			app.App.WithMcs()
		})
		c.AddFunc("0 * * * * *", func() {
			instance.ServerShellRun() // 每 1min 检查一次
		})
	}

	if app.App.Config.WithDingtalk.Enable {
		c.AddFunc("0 0 2 * * *", func() {
			err := dingtalk.LoadUsers()
			if err != nil {
				log.Error(err.Error())
			}
		})
		// 定时获取审批列表状态
		c.AddFunc("0 * * * * *", func() {
			dingtalk.LoadApproval()
		})
	}

	cron := "0 0 3 * * *" // 默认每天早上 3 点
	if app.App.Config.WithVideo.Cron != "" {
		cron = app.App.Config.WithVideo.Cron
	}
	c.AddFunc(cron, func() {
		sshd.AuditLogArchiver()
	})

	// 启动检测机器 ssh可连接性并依据配置发送钉钉告警通知
	if app.App.Config.WithSSHCheck.Enable {
		log.Infof("with ssh check,5min check once")
		c.AddFunc("*/5 * * * * *", func() {
			instance.ServerLiveness(app.App.Config.WithSSHCheck.Alert.RobotToken)
		})
	}

	c.Start()
	select {}
}
