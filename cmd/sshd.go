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
	"github.com/google/gops/agent"
	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron"
	"github.com/spf13/cobra"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/jump"
	"github.com/xops-infra/jms/core/sshd"
	"github.com/xops-infra/jms/io"
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
		if err := agent.Listen(agent.Options{}); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to start gops agent: %v\n", err)
			os.Exit(1)
		}
		defer agent.Close()

		appConfig.LoadYaml(config)

		err := os.MkdirAll(utils.FilePath(logDir), 0755)
		if err != nil {
			log.Fatalf("create log dir failed: %s", err.Error())
		}

		// init app
		_app := app.NewApp(debug, logDir, rootCmd.Version)

		if app.App.Config.WithLdap.Enable {
			log.Infof("enable ldap")
			_app.WithLdap()
		}

		if app.App.Config.WithDB.Enable {
			_app.WithDB(false) // 直管连接
			log.Infof("enable db")
		}

		app.App.Sshd.PolicyIO = io.NewPolicy(app.App.JmsDBService)
		app.App.Sshd.SshdIO = io.NewSshd(app.App.JmsDBService, app.App.Config.LocalServers.ToMapWithHost())
		app.App.Sshd.KeyIO = io.NewKey(app.App.JmsDBService)

		if !debug {
			go startSshdScheduler()
		}

		ssh.Handle(func(sess ssh.Session) {
			defer func() {
				if e, ok := recover().(error); ok {
					log.Errorf("sessionHandler panic: %s", e.Error())
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
		err := app.App.Sshd.Ldap.Login(ctx.User(), pass)
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
		log.Infof("user: %s, remote addr: %s login success", user, remote)
		sshHandler(sess)
	default:
		log.Infof("[default] user: %s, remote addr: %s login success", user, remote)
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

// 注意任务要做好分布式兼容
func startSshdScheduler() {

	c := cron.New()

	if app.App.Config.WithDB.Enable {
		c.AddFunc("0 * * * * *", func() {
			err := sshd.ServerShellRun() // 每 1min 检查一次
			if err != nil {
				log.Errorf("server shell run error: %s", err)
			}
		})
	}

	// 启动检测机器 ssh可连接性并依据配置发送钉钉告警通知
	if app.App.Config.WithSSHCheck.Enable {
		log.Infof("with ssh check,5min check once")
		c.AddFunc("0 */5 * * * *", func() {
			sshd.ServerLiveness(app.App.Config.WithSSHCheck.Alert.RobotToken)
		})
	}

	cron := "0 0 3 * * *" // 默认每天早上 3 点
	if app.App.Config.WithVideo.Cron != "" {
		cron = app.App.Config.WithVideo.Cron
	}
	c.AddFunc(cron, func() {
		sshd.AuditLogArchiver()
	})

	c.Start()
	select {}
}
