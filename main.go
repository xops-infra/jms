package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/elfgzp/ssh"
	"github.com/robfig/cron"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/instance"
	"github.com/xops-infra/jms/core/jump"
	"github.com/xops-infra/jms/core/sshd"
	"github.com/xops-infra/jms/utils"
)

var (
	port         int
	sshDir       string
	debug        bool
	logDir       string
	withSSHCheck bool
)

func init() {
	flag.StringVar(&sshDir, "ssh-dir", "~/.ssh/", "ssh dir")
	flag.IntVar(&port, "port", 22222, "ssh port")
	flag.BoolVar(&debug, "debug", false, "debug mode")
	flag.BoolVar(&withSSHCheck, "with-ssh-check", false, "with ssh check")
	flag.StringVar(&logDir, "log-dir", "/opt/logs/", "log file")
}

func passwordAuth(ctx ssh.Context, pass string) bool {
	err := app.App.Ldap.Login(ctx.User(), pass)
	return err == nil
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
	jps := jump.Service{}
	jps.Run(sess)
}

func scpHandler(args []string, sess *ssh.Session) {
	sshd.ExecuteSCP(args, sess)
}

func startScheduler() {
	c := cron.New()
	time.Sleep(10 * time.Second)
	c.AddFunc("*/50 * * * * *", func() {
		instance.LoadServer(app.App.Config)
	})

	c.Start()
	select {}
}

func main() {
	flag.Parse()

	// 处理～家目录不识别问题
	if strings.HasPrefix(sshDir, "~") {
		sshDir = strings.Replace(sshDir, "~", os.Getenv("HOME"), 1)
	}
	os.MkdirAll(utils.FilePath(logDir), 0755)
	// 判断文件hostAuthorizedKeys是否存在，不存在则创建
	hostAuthorizedKeys := sshDir + "authorized_keys"
	if !utils.FileExited(hostAuthorizedKeys) {
		// 600权限
		os.Create(hostAuthorizedKeys)
		os.Chmod(hostAuthorizedKeys, 0600)
	}
	if debug {
		log.Default().WithLevel(log.DebugLevel).WithHumanTime(time.Local).WithFilename(logDir + "app.log").Init()
		log.Debugf("debug mode, disabled scheduler")
	} else {
		log.Default().WithLevel(log.InfoLevel).WithHumanTime(time.Local).WithFilename(logDir + "app.log").Init()
		go startScheduler()
	}
	hostKeyFile := sshDir + "id_rsa"
	// log.Panicf(hostKeyFile)
	if !utils.FileExited(hostKeyFile) {
		sshd.GenKey(hostKeyFile)
	}

	app.NewApplication(debug, sshDir).WithDingTalk()

	instance.LoadServer(app.App.Config)

	// 启动检测机器 ssh可连接性并依据配置发送钉钉告警通知
	if withSSHCheck {
		log.Infof("with ssh check,5min check once")
		go func() {
			for {
				instance.ServerLiveness(app.App.Config.DingTalk.RobotToken)
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

	log.Infof("starting ssh server on port %d...\n", port)
	err := ssh.ListenAndServe(
		fmt.Sprintf(":%d", port),
		nil,
		ssh.PasswordAuth(passwordAuth),
		ssh.PublicKeyAuth(publicKeyAuth),
		ssh.HostKeyFile(utils.FilePath(hostKeyFile)),
	)
	if err != nil {
		log.Panic(err.Error())
	}
}
