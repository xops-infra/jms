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
	"github.com/anmitsu/go-shlex"
	"github.com/elfgzp/ssh"
	"github.com/fatih/color"
	"github.com/helloyi/go-sshclient"
	"github.com/patrickmn/go-cache"
	"github.com/xops-infra/noop/log"
	gossh "golang.org/x/crypto/ssh"

	"github.com/xops-infra/jms/app"
	. "github.com/xops-infra/jms/model"
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

// new audit log
func newAuditLog(user, host string) (*os.File, error) {
	auditDir := app.App.Config.WithVideo.Dir
	logFile := fmt.Sprintf("%s/%s_%s_%s.log", auditDir, time.Now().Format("20060102_150405"), host, user)
	logIo, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return logIo, nil
}

// 独立的阻塞远程客户端连接方法
func NewTerminal(server Server, sshUser SSHUser, sess *ssh.Session) error {
	log.Infof("NewTerminal server: %v, sshUser: %v, user: %v", server.Host, sshUser.KeyName, (*sess).User())
	proxyClient, upstreamClient, actual, err := ConnectSSHClient((*sess).User(), server, sshUser)
	if err != nil {
		log.Errorf("NewSSHClient error: %s", err)
		return err
	}
	if proxyClient != nil {
		defer proxyClient.Close()
	}
	defer upstreamClient.Close()
	if app.App.Config.WithDB.Enable {
		actor, client, host, id := (*sess).User(), (*sess).RemoteAddr().String(), server.Host, server.ID
		if err := app.App.DBIo.AddServerLoginRecord(&AddSshLoginRequest{
			User: &actor, Client: &client, TargetServer: &host, InstanceID: &id,
			Source: AuditSourceSSH, SSHUser: actual.UserName, SSHKey: actual.KeyName,
		}); err != nil {
			log.Errorf("create ssh login record error: %v", err)
		}
	}

	if sshUser.Auto {
		Info(fmt.Sprintf("已使用 %s 连接", actual.UserName), sess)
	}

	upstreamSess, err := upstreamClient.NewSession()
	if err != nil {
		return err
	}
	defer upstreamSess.Close()

	var writer io.Writer

	if app.App.Config.WithVideo.Enable {
		// 创建日志文件
		logFile, err := newAuditLog((*sess).User(), server.Host)
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

	// fmt.Println((*sess).Environ(), (*sess).RemoteAddr())
	app.App.Sshd.UserCache.Set((*sess).RemoteAddr().String(), true, cache.DefaultExpiration)
	defer app.App.Sshd.UserCache.Delete((*sess).RemoteAddr().String())

	if err := upstreamSess.Wait(); err != nil {
		return err
	}

	return nil
}

// 判断服务器是否配置了代理，配置获取方式可以是本地，或者数据库
// 本地配置的优先级高于数据库配置
func isProxyServer(server Server) (*CreateProxyRequest, error) {
	if app.App.Config.WithDB.Enable {
		proxys, err := app.App.DBIo.ListProxy()
		if err != nil {
			return nil, err
		}
		for _, proxy := range proxys {
			if strings.HasPrefix(server.Host, *proxy.IPPrefix) {
				log.Debugf("get proxy from config for %s: host=%s port=%d user=%s", server.Host, tea.StringValue(proxy.Host), tea.IntValue(proxy.Port), tea.StringValue(proxy.LoginUser))
				return &proxy, nil
			}
		}
		log.Debugf("no proxy found: %s", server.Host)
		return nil, nil
	} else {
		// 本地配置
		return nil, nil
	}
}

// NewSSHClient NewSSHClient
// proxy client 返回主要是为了外部 close 用。
func newSSHClient(user string, server Server, sshUser SSHUser) (*gossh.Client, *gossh.Client, error) {
	proxy, err := isProxyServer(server)
	if err != nil {
		return nil, nil, fmt.Errorf("SSH proxy lookup failed (stage=proxy_lookup target=%q): %w", net.JoinHostPort(server.Host, fmt.Sprint(server.Port)), err)
	}
	if proxy != nil {
		return ProxyClient(server, *proxy, sshUser)
	}
	log.Infof("%s direct connect: %s:%d", user, server.Host, server.Port)
	address := net.JoinHostPort(server.Host, fmt.Sprint(server.Port))
	config, err := newSshConfig(sshUser)
	if err != nil {
		return nil, nil, fmt.Errorf("SSH target credentials failed (stage=target_credentials target=%q user=%q key=%q): %w", address, sshUser.UserName, sshUser.KeyName, err)
	}
	client, err := gossh.Dial("tcp", address, config)
	if err != nil {
		return nil, nil, &ConnectionError{Stage: "target_connect", Detail: fmt.Sprintf("target=%q user=%q key=%q", address, sshUser.UserName, sshUser.KeyName), Err: err}
	}
	return nil, client, nil
}

func newSshConfig(sshUser SSHUser) (*gossh.ClientConfig, error) {
	config := &gossh.ClientConfig{
		User:            sshUser.UserName,
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
		signer, err := getSignerFromLocal(app.App.SSHDir + strings.TrimPrefix(sshUser.KeyName, "/"))
		if err != nil {
			return nil, err
		}
		config.Auth = append(config.Auth, gossh.PublicKeys(signer))
	} else {
		return nil, fmt.Errorf("server login user auth not set, please check password or private key for %s", sshUser.UserName)
	}
	return config, nil
}

func ProxyClient(instance Server, proxy CreateProxyRequest, sshUser SSHUser) (*gossh.Client, *gossh.Client, error) {
	if proxy.LoginUser == nil || *proxy.LoginUser == "" || tea.StringValue(proxy.Host) == "" || tea.IntValue(proxy.Port) == 0 {
		return nil, nil, fmt.Errorf("proxy config error (stage=proxy_config): host=%q port=%d user=%q", tea.StringValue(proxy.Host), tea.IntValue(proxy.Port), tea.StringValue(proxy.LoginUser))
	}
	proxyAddress := net.JoinHostPort(*proxy.Host, fmt.Sprint(*proxy.Port))
	targetAddress := net.JoinHostPort(instance.Host, fmt.Sprint(instance.Port))
	proxyAuth := "password"
	// 支持密码或者私钥认证
	proxyConfig := &gossh.ClientConfig{
		User:            *proxy.LoginUser,
		HostKeyCallback: gossh.HostKeyCallback(func(hostname string, remote net.Addr, key gossh.PublicKey) error { return nil }),
		Timeout:         8 * time.Second,
	}
	if proxy.LoginPasswd != nil && *proxy.LoginPasswd != "" {
		log.Debugf("proxy authentication method: password")
		proxyConfig.Auth = append(proxyConfig.Auth, gossh.Password(*proxy.LoginPasswd))
	} else if proxy.KeyID != nil && *proxy.KeyID != "" {
		proxyAuth = "key_id=" + *proxy.KeyID
		// 走 proxy keyID 去获取认证信息
		log.Debugf("proxy keyID: %s", *proxy.KeyID)
		signerProxy, err := app.App.Sshd.SshdIO.GetSignerByKeyID(*proxy.KeyID)
		if err != nil {
			return nil, nil, fmt.Errorf("SSH proxy credentials failed (stage=proxy_credentials proxy=%q user=%q auth=%q): %w", proxyAddress, *proxy.LoginUser, proxyAuth, err)
		}
		proxyConfig.Auth = append(proxyConfig.Auth, gossh.PublicKeys(signerProxy))
	} else if proxy.IdentityFile != nil && *proxy.IdentityFile != "" {
		proxyAuth = "key_file=" + *proxy.IdentityFile
		// 兼容数据库通过 identityFile 认证, 随后走文件认证
		signerProxy, err := app.App.Sshd.SshdIO.GetSignerByIdentityFile(*proxy.IdentityFile)
		if err != nil {
			// 走文件认证
			log.Debugf("proxy identityFile: %s", *proxy.IdentityFile)
			_signerProxy, err := getSignerFromLocal(app.App.SSHDir + strings.TrimPrefix(*proxy.IdentityFile, "/"))
			if err != nil {
				return nil, nil, fmt.Errorf("SSH proxy credentials failed (stage=proxy_credentials proxy=%q user=%q auth=%q): %w", proxyAddress, *proxy.LoginUser, proxyAuth, err)
			}
			signerProxy = _signerProxy
		}
		proxyConfig.Auth = append(proxyConfig.Auth, gossh.PublicKeys(signerProxy))
	} else {
		return nil, nil, fmt.Errorf("proxy config error has no auth (stage=proxy_config): host=%q port=%d user=%q", tea.StringValue(proxy.Host), tea.IntValue(proxy.Port), tea.StringValue(proxy.LoginUser))
	}
	log.Infof("SSH route target=%q target_user=%q target_key=%q proxy=%q proxy_user=%q proxy_auth=%q", targetAddress, sshUser.UserName, sshUser.KeyName, proxyAddress, *proxy.LoginUser, proxyAuth)
	proxyClient, err := gossh.Dial("tcp", proxyAddress, proxyConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("SSH proxy connection failed (stage=proxy_connect proxy=%q user=%q auth=%q target=%q): %w", proxyAddress, *proxy.LoginUser, proxyAuth, targetAddress, err)
	}
	// Transfer ownership to the caller only after both handshakes succeed.
	connected := false
	defer func() {
		if !connected {
			proxyClient.Close()
		}
	}()

	conn, err := proxyClient.Dial("tcp", targetAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("SSH proxy forwarding failed (stage=proxy_forward proxy=%q target=%q): %w", proxyAddress, targetAddress, err)
	}
	defer func() {
		if !connected {
			conn.Close()
		}
	}()

	config, err := newSshConfig(sshUser)
	if err != nil {
		return nil, nil, fmt.Errorf("SSH target credentials failed (stage=target_credentials target=%q user=%q key=%q proxy=%q): %w", targetAddress, sshUser.UserName, sshUser.KeyName, proxyAddress, err)
	}
	clientConn, proxyChans, proxyReqs, err := gossh.NewClientConn(conn, targetAddress, config)
	if err != nil {
		return nil, nil, &ConnectionError{Stage: "target_connect", Detail: fmt.Sprintf("target=%q user=%q key=%q proxy=%q", targetAddress, sshUser.UserName, sshUser.KeyName, proxyAddress), Err: err}
	}
	client := gossh.NewClient(clientConn, proxyChans, proxyReqs)
	connected = true
	return proxyClient, client, nil
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
	command = strings.TrimSpace(command)
	if command == "" {
		return "", nil, errors.New("No command in payload: " + command)
	}

	parts, err := shlex.Split(command, true)
	if err != nil {
		return "", nil, err
	}

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
