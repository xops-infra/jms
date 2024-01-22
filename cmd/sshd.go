package cmd

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/elfgzp/ssh"
	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron"
	"github.com/spf13/cobra"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	appConfig "github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/core/dingtalk"
	"github.com/xops-infra/jms/core/instance"
	"github.com/xops-infra/jms/core/jump"
	"github.com/xops-infra/jms/core/policy"
	"github.com/xops-infra/jms/core/sshd"
	"github.com/xops-infra/jms/utils"
)

var (
	sshDir   string
	logDir   string
	timeOut  int // s
	sshdPort int
)

var sshdCmd = &cobra.Command{
	Use:   "sshd",
	Short: "A brief description of your application",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		appConfig.Load(config)
		log.Default().WithLevel(log.InfoLevel).WithHumanTime(time.Local).WithFilename(strings.TrimSuffix(logDir, "/") + "/sshd.log").Init()
		log.Infof("config file: %s", config)
		if debug {
			log.Default().WithLevel(log.DebugLevel).WithHumanTime(time.Local).WithFilename(strings.TrimSuffix(logDir, "/") + "/sshd.log").Init()
			log.Debugf("debug mode, disabled scheduler")
		} else {
			go startScheduler()
		}

		// 处理～家目录不识别问题
		if strings.HasPrefix(sshDir, "~") {
			sshDir = strings.Replace(sshDir, "~", os.Getenv("HOME"), 1)
		}
		err := os.MkdirAll(utils.FilePath(logDir), 0755)
		if err != nil {
			log.Fatalf("create log dir failed: %s", err.Error())
		}

		// 判断文件hostAuthorizedKeys是否存在，不存在则创建
		hostAuthorizedKeys := sshDir + "authorized_keys"
		if !utils.FileExited(hostAuthorizedKeys) {
			// 600权限
			os.Create(hostAuthorizedKeys)
			os.Chmod(hostAuthorizedKeys, 0600)
		}
		hostKeyFile := sshDir + "id_rsa"
		// log.Panicf(hostKeyFile)
		if !utils.FileExited(hostKeyFile) {
			sshd.GenKey(hostKeyFile)
		}

		_app := app.NewSshdApplication(debug, sshDir)

		if app.App.Config.WithSSHCheck.Enable {
			log.Infof("enable dingtalk")
			_app.WithRobot()
		}

		if app.App.Config.WithDingtalk.Enable {
			log.Infof("enable dingtalk")
			_app.WithDingTalk()
		}

		if app.App.Config.WithPolicy.Enable {
			_app.WithPolicy()
			log.Infof("enable policy,default user: admin/admin")
		} else {
			log.Infof("--with-policy=false, this mode any server can be connected")
		}

		if app.App.Config.WithDingtalk.Enable {
			if !app.App.Config.WithPolicy.Enable {
				log.Panicf("with-api-server-approval must be used with --with-policy=true")
			}
			log.Infof("enable api dingtalk Approve")
		}

		instance.LoadServer(app.App.Config)

		// 启动检测机器 ssh可连接性并依据配置发送钉钉告警通知
		if app.App.Config.WithSSHCheck.Enable {
			log.Infof("with ssh check,5min check once")
			go func() {
				for {
					instance.ServerLiveness(app.App.Config.WithSSHCheck.Alert.RobotToken)
					time.Sleep(5 * time.Minute)
				}
			}()
		}

		ssh.Handle(func(sess ssh.Session) {
			defer func() {
				if e, ok := recover().(error); ok {
					log.Error(e.Error())
				}
			}()
			sessionHandler(&sess)
		})

		var wrapped *wrappedConn

		log.Infof("starting ssh server on port %d...\n", sshdPort)
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

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.jms.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.AddCommand(sshdCmd)
	sshdCmd.Flags().StringVar(&sshDir, "ssh-dir", "~/.ssh/", "ssh dir")
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
	_, err := app.App.PolicyService.Login(ctx.User(), pass)
	if err != nil {
		log.Error(err.Error())
		return false
	}
	return true
}

func publicKeyAuth(ctx ssh.Context, key ssh.PublicKey) bool {
	hostAuthorizedKeys := sshDir + "authorized_keys"
	data, err := os.ReadFile(hostAuthorizedKeys)
	if err != nil {
		log.Error(err.Error())
		return false
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, ctx.User()) {
			allowed, _, _, _, _ := ssh.ParseAuthorizedKey([]byte(line))
			if ssh.KeysEqual(key, allowed) {
				log.Debugf("user: %s, pub: %s", ctx.User(), line)
				return true
			}
		}
	}
	return false
}

func sessionHandler(sess *ssh.Session) {
	defer func() {
		(*sess).Close()
	}()
	user := (*sess).User()
	remote := (*sess).RemoteAddr()
	_, found := app.App.UserCache.Get(user)
	if !found {
		app.App.UserCache.Add(user, 1, cache.DefaultExpiration)
	}
	// 如果启用 policy策略，默认开始注册登录用户入库
	if app.App.Config.WithPolicy.Enable {
		_, err := app.App.PolicyService.CreateUser(&policy.UserMut{
			Username: &user,
		})
		if err != nil {
			if !strings.Contains(err.Error(), "user already exists") {
				log.Error(err.Error())
			}
		} else {
			msg := fmt.Sprintf("首次登录，用户信息%s已入库！组信息请联系管理员维护", user)
			log.Infof(msg)
			sshd.Info(msg, sess)
		}

	}
	log.Infof("user: %s, remote addr: %s login success", user, remote)
	rawCmd := (*sess).RawCommand()
	cmd, args, err := sshd.ParseRawCommand(rawCmd)
	if err != nil {
		sshd.ErrorInfo(err, sess)
		return
	}
	log.Debugf("cmd: %s, args: %s\n", cmd, args)
	switch cmd {
	case "exec":
		execHandler(args, sess)
	case "scp":
		scpHandler(args, sess)
	default:
		if strings.Contains(cmd, "umask") {
			// 版本问题导致的 cmd不一致问题
			execHandler(args, sess)
		}
		sshHandler(sess)
	}
}

func execHandler(args []string, sess *ssh.Session) {
	// 执行命令
	// 获取用户后续输入的 pubKey 存放到 authorized_keys 文件中
	pubKey, err := bufio.NewReader(*sess).ReadString('\n')
	if err != nil {
		log.Error(err.Error())
		return
	}
	if !strings.Contains(pubKey, "ssh-rsa") {
		sshd.ErrorInfo(fmt.Errorf("pub key already exists"), sess)
		return
	}
	hostAuthorizedKeys := sshDir + "authorized_keys"
	data, err := os.ReadFile(hostAuthorizedKeys)
	if err != nil {
		log.Error(err.Error())
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.Contains(line, pubKey) {
			sshd.Info("pub key already exists", sess)
			return
		}
	}
	// 将公钥添加到authorized_keys的第一行
	f, err := os.OpenFile(hostAuthorizedKeys, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		log.Error(err.Error())
		return
	}
	defer f.Close()
	_, err = f.WriteString((*sess).User() + " " + pubKey + "\n" + string(data))
	if err != nil {
		log.Error(err.Error())
		return
	}
	log.Infof("add pub key: %s to %s success", pubKey, hostAuthorizedKeys)
	// 退出
	(*sess).Exit(0)
}

func sshHandler(sess *ssh.Session) {
	jps := jump.NewService(sess, time.Duration(timeOut)*time.Second)
	jps.Run()
}

func scpHandler(args []string, sess *ssh.Session) {
	sshd.ExecuteSCP(args, sess)
}

func startScheduler() {
	c := cron.New()
	time.Sleep(10 * time.Second) // 等待app初始化完成
	c.AddFunc("0 */2 * * * *", func() {
		instance.LoadServer(app.App.Config)
	})
	if app.App.Config.WithDingtalk.Enable {
		c.AddFunc("0 0 2 * * *", func() {
			err := dingtalk.LoadUsers()
			if err != nil {
				log.Error(err.Error())
			}
		})
		c.AddFunc("0 * * * * *", func() {
			dingtalk.LoadApproval()
		})
	}
	if app.App.Config.APPSet.Audit.Enable {
		log.Infof("enabled audit log archiver,config: %s", tea.Prettify(app.App.Config.APPSet.Audit))
		cron := "0 0 3 * * *"
		if app.App.Config.APPSet.Audit.Cron != "" {
			cron = app.App.Config.APPSet.Audit.Cron
		}
		c.AddFunc(cron, func() {
			sshd.AuditLogArchiver()
		})
	}

	c.Start()
	select {}
}
