package sshd

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/elfgzp/ssh"
	"github.com/fatih/color"
	"github.com/helloyi/go-sshclient"
	"github.com/patrickmn/go-cache"
	"github.com/xops-infra/noop/log"
	gossh "golang.org/x/crypto/ssh"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/jms/utils"
)

// GetClientByPasswd GetClientByPasswd
func GetClientByPasswd(username, host string, port int, passwd string) (*sshclient.Client, error) {
	client, err := sshclient.DialWithPasswd(
		fmt.Sprintf("%s:%d", host, port),
		username,
		passwd,
	)

	if err != nil {
		return nil, err
	}
	return client, nil
}

// NewTerminal NewTerminal
func NewTerminal(server config.Server, sshUser config.SSHUser, sess *ssh.Session, timeout string) error {
	proxyClient, upstreamClient, err := NewSSHClient(server, sshUser)
	if err != nil {
		log.Errorf("NewSSHClient error: %s", err)
		return err
	}
	if proxyClient != nil {
		defer proxyClient.Close()
	}

	upstreamSess, err := upstreamClient.NewSession()
	if err != nil {
		return err
	}
	defer upstreamSess.Close()

	var writer io.Writer

	if app.App.Config.WithVideo.Enable {
		// 创建日志文件
		logFile, err := NewAuditLog((*sess).User(), server.Host)
		if err != nil {
			return err
		}
		defer logFile.Close()
		writer = io.MultiWriter(logFile, *sess)
	} else {
		writer = *sess
	}

	// 发送屏幕清理指令
	// (*sess).Write([]byte("\033c"))

	// 创建同时写入日志文件和终端的写入器
	upstreamSess.Stdout = writer
	upstreamSess.Stdin = *sess
	upstreamSess.Stderr = writer

	pty, winCh, _ := (*sess).Pty()

	if err := upstreamSess.RequestPty(pty.Term, pty.Window.Height, pty.Window.Width, pty.TerminalModes); err != nil {
		return err
	}

	if err := upstreamSess.Shell(); err != nil {
		return err
	}

	go func() {
		for win := range winCh {
			upstreamSess.WindowChange(win.Height, win.Width)
		}
	}()
	fmt.Println((*sess).Environ(), (*sess).RemoteAddr())
	err = app.App.Cache.Add((*sess).RemoteAddr().String(), true, cache.DefaultExpiration)
	if err != nil {
		log.Errorf("add cache error: %s", err)
	}
	defer app.App.Cache.Delete((*sess).RemoteAddr().String())

	if err := upstreamSess.Wait(); err != nil {
		return err
	}

	return nil
}

// 判断服务器是否配置了代理，配置获取方式可以是本地，或者数据库
// 本地配置的优先级高于数据库配置
func isProxyServer(server config.Server) (*db.CreateProxyRequest, error) {
	for _, proxy := range app.App.Config.Proxys {
		log.Debugf("host %s proxy: %s\n", server.Host, tea.Prettify(proxy))
		if strings.HasPrefix(server.Host, *proxy.IPPrefix) {
			log.Debugf("get proxy from config for %s, %s\n", server.Host, tea.Prettify(proxy))
			return &proxy, nil
		}
	}
	log.Debugf("no proxy found: %s\n", server.Host)
	return nil, nil
}

// NewSSHClient NewSSHClient
func NewSSHClient(server config.Server, sshUser config.SSHUser) (*gossh.Client, *gossh.Client, error) {
	proxy, err := isProxyServer(server)
	if err != nil {
		return nil, nil, err
	}
	if proxy != nil {
		return ProxyClient(server, *proxy, sshUser)
	}
	log.Debugf("direct connect: %s:%d\n", server.Host, server.Port)
	config, err := newSshConfig(sshUser)
	if err != nil {
		return nil, nil, err
	}
	client, err := gossh.Dial("tcp", fmt.Sprintf("%s:%d", server.Host, server.Port), config)
	return nil, client, err
}

func newSshConfig(sshUser config.SSHUser) (*gossh.ClientConfig, error) {
	config := &gossh.ClientConfig{
		User:            sshUser.SSHUsername,
		HostKeyCallback: gossh.HostKeyCallback(func(hostname string, remote net.Addr, key gossh.PublicKey) error { return nil }),
		Timeout:         8 * time.Second,
	}
	// 优先密码认证，其次私钥认证
	if sshUser.Password != "" {
		config.Auth = append(config.Auth, gossh.Password(sshUser.Password))
	} else if sshUser.Base64Pem != "" {
		signer, err := getSignerFromBase64(sshUser.Base64Pem)
		if err != nil {
			return nil, err
		}
		config.Auth = append(config.Auth, gossh.PublicKeys(signer))
	} else if sshUser.KeyName != "" {
		signer, err := getSignerFromLocal(strings.TrimSuffix(app.App.SshDir, "/") + "/" + strings.TrimPrefix(sshUser.KeyName, "/"))
		if err != nil {
			return nil, err
		}
		config.Auth = append(config.Auth, gossh.PublicKeys(signer))
	} else {
		return nil, fmt.Errorf("server login user auth not set, please check password or private key for %s", sshUser.SSHUsername)
	}
	return config, nil
}

func ProxyClient(instance config.Server, proxy db.CreateProxyRequest, sshUser config.SSHUser) (*gossh.Client, *gossh.Client, error) {
	if proxy.LoginUser == nil || *proxy.LoginUser == "" || tea.StringValue(proxy.Host) == "" || tea.IntValue(proxy.Port) == 0 {
		return nil, nil, fmt.Errorf("proxy config error, %s", tea.Prettify(proxy))
	}
	// 支持密码或者私钥认证
	proxyConfig := &gossh.ClientConfig{
		User:            *proxy.LoginUser,
		HostKeyCallback: gossh.HostKeyCallback(func(hostname string, remote net.Addr, key gossh.PublicKey) error { return nil }),
		Timeout:         8 * time.Second,
	}
	if proxy.LoginPasswd != nil && *proxy.LoginPasswd != "" {
		log.Debugf("proxy login passwd: %s", *proxy.LoginPasswd)
		proxyConfig.Auth = append(proxyConfig.Auth, gossh.Password(*proxy.LoginPasswd))
	} else if proxy.IdentityFile != nil {
		log.Debugf("proxy identity file: %s", *proxy.IdentityFile)
		signerProxy, err := getSigner(*proxy.IdentityFile)
		if err != nil {
			return nil, nil, err
		}
		proxyConfig.Auth = append(proxyConfig.Auth, gossh.PublicKeys(signerProxy))
	}
	log.Infof("connecting %s with proxy connect: %s:%d", instance.Host, *proxy.Host, *proxy.Port)
	proxyClient, err := gossh.Dial("tcp", fmt.Sprintf("%s:%d", *proxy.Host, *proxy.Port), proxyConfig)
	if err != nil {
		return nil, nil, err
	}

	conn, err := proxyClient.Dial("tcp", fmt.Sprintf("%s:%d", instance.Host, instance.Port))
	if err != nil {
		return nil, nil, err
	}

	config, err := newSshConfig(sshUser)
	if err != nil {
		return nil, nil, err
	}
	clientConn, proxyChans, proxyReqs, err := gossh.NewClientConn(conn, fmt.Sprintf("%s:%d", instance.Host, instance.Port), config)
	if err != nil {
		return nil, nil, err
	}
	client := gossh.NewClient(clientConn, proxyChans, proxyReqs)

	return proxyClient, client, nil
}

// 在 key里面获取签名，支持数据库 base64 或者本地文件
func getSigner(identityFile string) (gossh.Signer, error) {
	if key, ok := app.App.Config.Keys.ToMapWithName()[identityFile]; ok {
		if key.PemBase64 != nil {
			log.Debugf("got pem base64 for %s", identityFile)
			return getSignerFromBase64(*key.PemBase64)
		}
		if key.IdentityFile != nil {
			log.Debugf("got pem file for %s", identityFile)
			return getSignerFromLocal(*key.IdentityFile)
		}
	}
	return nil, fmt.Errorf("key %s not found", identityFile)
}

func getSignerFromLocal(identityFile string) (gossh.Signer, error) {
	log.Debugf("identityFile: %s\n", identityFile)
	key, err := os.ReadFile(utils.FilePath(identityFile))
	if err != nil {
		return nil, err
	}
	signer, err := gossh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}
	return signer, nil
}

func getSignerFromBase64(key string) (gossh.Signer, error) {
	// bas64 decode
	base64Pem, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, err
	}
	signer, err := gossh.ParsePrivateKey([]byte(base64Pem))
	if err != nil {
		return nil, err
	}
	return signer, nil
}

// ParseRawCommand ParseRawCommand
func ParseRawCommand(command string) (string, []string, error) {
	parts := strings.Split(command, " ")
	if len(parts) < 1 {
		return "", nil, errors.New("No command in payload: " + command)
	}

	if len(parts) < 2 {
		return parts[0], []string{}, nil
	}

	return parts[0], parts[1:], nil
}

// ErrorInfo ErrorInfo
func ErrorInfo(err error, sess *ssh.Session) {
	read := color.New(color.FgRed)
	read.Fprint(*sess, fmt.Sprintf("%s\n", err))
}

// Info Info
func Info(msg string, sess *ssh.Session) {
	green := color.New(color.FgGreen)
	green.Fprint(*sess, fmt.Sprintf("%s\n", msg))
}
