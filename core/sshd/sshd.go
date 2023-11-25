package sshd

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/elfgzp/ssh"
	"github.com/fatih/color"
	"github.com/helloyi/go-sshclient"
	"github.com/patsnapops/noop/log"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/config"
	"github.com/xops-infra/jms/core/trzsz"
	"github.com/xops-infra/jms/utils"
	gossh "golang.org/x/crypto/ssh"
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
func NewTerminal(server config.Server, sshUser *config.SSHUser, sess *ssh.Session) error {

	proxyClient, upstreamClient, err := NewSSHClient(server, sshUser)
	if err != nil {
		log.Errorf("NewSSHClient error: %s", err)
		return err
	}
	defer proxyClient.Close()

	upstreamSess, err := upstreamClient.NewSession()
	if err != nil {
		return err
	}
	defer upstreamSess.Close()

	// 创建日志文件
	logFile, err := NewAuditLog((*sess).User(), server.Host)
	if err != nil {
		return err
	}
	defer logFile.Close()

	// 创建同时写入日志文件和终端的写入器
	writer := io.MultiWriter(logFile, *sess)
	upstreamSess.Stdout = writer
	upstreamSess.Stdin = *sess
	upstreamSess.Stderr = writer

	if false {
		// TODO 未完成
		serverIn, err := upstreamSess.StdinPipe()
		if err != nil {
			return err
		}
		serverOut, err := upstreamSess.StdoutPipe()
		if err != nil {
			return err
		}

		if err := trzsz.WithTrzsz(*sess, nil, upstreamClient, upstreamSess, serverIn, serverOut); err != nil {
			return err
		}
	}

	pty, winCh, _ := (*sess).Pty()

	if err := upstreamSess.RequestPty(pty.Term, pty.Window.Height, pty.Window.Width, pty.TerminalModes); err != nil {
		return err
	}

	// TODO: 命令录制审计；
	if err := upstreamSess.Shell(); err != nil {
		return err
	}

	go func() {
		for win := range winCh {
			upstreamSess.WindowChange(win.Height, win.Width)
		}
	}()

	if err := upstreamSess.Wait(); err != nil {
		return err
	}

	return nil
}

// NewSSHClient NewSSHClient
func NewSSHClient(server config.Server, sshUser *config.SSHUser) (*gossh.Client, *gossh.Client, error) {

	if server.Proxy != nil {
		log.Debugf("get proxy: %s:%d\n", server.Proxy.Host, server.Proxy.Port)
		return ProxyClient(server, sshUser)
	}
	log.Debugf("direct connect: %s:%d\n", server.Host, server.Port)
	signer, err := geSigner(strings.TrimSuffix(app.App.SshDir, "/") + "/" + strings.TrimPrefix(sshUser.IdentityFile, "/"))
	if err != nil {
		return nil, nil, err
	}
	config := &gossh.ClientConfig{
		User: sshUser.SSHUsername,
		Auth: []gossh.AuthMethod{
			gossh.PublicKeys(signer),
		},
		HostKeyCallback: gossh.HostKeyCallback(func(hostname string, remote net.Addr, key gossh.PublicKey) error { return nil }),
		Timeout:         8 * time.Second,
	}
	client, err := gossh.Dial("tcp", fmt.Sprintf("%s:%d", server.Host, server.Port), config)
	return nil, client, err
}

func ProxyClient(instance config.Server, sshUser *config.SSHUser) (*gossh.Client, *gossh.Client, error) {
	signerProxy, err := geSigner(strings.TrimSuffix(app.App.SshDir, "/") + "/" + strings.TrimPrefix(instance.Proxy.SSHUsers.IdentityFile, "/"))
	if err != nil {
		return nil, nil, err
	}
	proxyConfig := &gossh.ClientConfig{
		User: instance.Proxy.SSHUsers.SSHUsername,
		Auth: []gossh.AuthMethod{
			gossh.PublicKeys(signerProxy),
		},
		HostKeyCallback: gossh.HostKeyCallback(func(hostname string, remote net.Addr, key gossh.PublicKey) error { return nil }),
		Timeout:         8 * time.Second,
	}
	proxyClient, err := gossh.Dial("tcp", fmt.Sprintf("%s:%d", instance.Proxy.Host, instance.Proxy.Port), proxyConfig)
	if err != nil {
		return nil, nil, err
	}

	signer, err := geSigner(strings.TrimSuffix(app.App.SshDir, "/") + "/" + strings.TrimPrefix(sshUser.IdentityFile, "/"))
	if err != nil {
		log.Errorf("signer error: %s", err)
		return nil, nil, err
	}
	config := &gossh.ClientConfig{
		User: sshUser.SSHUsername,
		Auth: []gossh.AuthMethod{
			gossh.PublicKeys(signer),
		},
		HostKeyCallback: gossh.HostKeyCallback(func(hostname string, remote net.Addr, key gossh.PublicKey) error { return nil }),
		Timeout:         8 * time.Second,
	}

	conn, err := proxyClient.Dial("tcp", fmt.Sprintf("%s:%d", instance.Host, instance.Port))
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

func geSigner(identityFile string) (gossh.Signer, error) {
	log.Debugf("identityFile: %s\n", identityFile)
	key, err := ioutil.ReadFile(utils.FilePath(identityFile))
	if err != nil {
		return nil, err
	}
	signer, err := gossh.ParsePrivateKey(key)
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
	green.Fprint(*sess, msg)
}
