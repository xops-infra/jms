package sshd

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
	gossh "golang.org/x/crypto/ssh"
)

// Exercise real SSH handshakes so proxy and target failures cannot be confused.
func TestProxyClientFailureStages(t *testing.T) {
	log.Default().WithFilename(filepath.Join(t.TempDir(), "ssh.log")).WithLevel(log.FatalLevel).Init()
	for _, stage := range []string{"proxy_connect", "proxy_forward", "target_credentials", "target_connect", "success"} {
		t.Run(stage, func(t *testing.T) {
			target := startSSHFixture(t, stage == "success", nil)
			proxyAddress := startSSHFixture(t, stage != "proxy_connect", func(ch gossh.NewChannel) {
				if stage == "proxy_forward" {
					ch.Reject(gossh.Prohibited, "forwarding disabled")
					return
				}
				upstream, err := net.Dial("tcp", target)
				if err != nil {
					ch.Reject(gossh.ConnectionFailed, "target unavailable")
					return
				}
				defer upstream.Close()
				channel, requests, err := ch.Accept()
				if err != nil {
					return
				}
				defer channel.Close()
				go gossh.DiscardRequests(requests)
				go func() { io.Copy(upstream, channel); upstream.Close() }()
				io.Copy(channel, upstream)
			})
			host, portString, _ := net.SplitHostPort(proxyAddress)
			var port int
			fmt.Sscan(portString, &port)
			targetHost, targetPortString, _ := net.SplitHostPort(target)
			var targetPort int
			fmt.Sscan(targetPortString, &targetPort)
			server := model.Server{Host: targetHost, Port: targetPort}
			user := model.SSHUser{UserName: "target-user", Password: "do-not-log-target-password", KeyName: "target-key-label"}
			if stage == "target_credentials" {
				user.Password = ""
				user.KeyName = ""
			}
			proxy := model.CreateProxyRequest{Host: tea.String(host), Port: tea.Int(port), LoginUser: tea.String("proxy-user"), LoginPasswd: tea.String("do-not-log-proxy-password")}
			proxyClient, client, err := ProxyClient(server, proxy, user)
			if stage == "success" {
				if err != nil {
					t.Fatal(err)
				}
				defer proxyClient.Close()
				defer client.Close()
				if _, _, err := client.SendRequest("keepalive@test", true, nil); err != nil {
					t.Fatalf("successful connection closed: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), "stage="+stage) {
				t.Fatalf("expected %s, got %v", stage, err)
			}
			if proxyClient != nil || client != nil {
				t.Fatal("failure returned a partial client")
			}
			for _, secret := range []string{user.Password, *proxy.LoginPasswd} {
				if secret != "" && strings.Contains(err.Error(), secret) {
					t.Fatal("error leaked a password")
				}
			}
			if errors.Unwrap(err) == nil {
				t.Fatal("underlying error was discarded")
			}
			if !strings.Contains(err.Error(), proxyAddress) {
				t.Fatalf("missing proxy context: %v", err)
			}
			if stage == "target_connect" && (!strings.Contains(err.Error(), "target-user") || !strings.Contains(err.Error(), "target-key-label")) {
				t.Fatalf("missing target identity: %v", err)
			}
		})
	}
}

func startSSHFixture(t *testing.T, allow bool, handle func(gossh.NewChannel)) string {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := gossh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	config := &gossh.ServerConfig{PasswordCallback: func(gossh.ConnMetadata, []byte) (*gossh.Permissions, error) {
		if !allow {
			return nil, errors.New("fixture authentication rejected")
		}
		return nil, nil
	}}
	config.AddHostKey(signer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { listener.Close() })
	go func() {
		for {
			raw, err := listener.Accept()
			if err != nil {
				return
			}
			go func() {
				defer raw.Close()
				raw.SetDeadline(time.Now().Add(5 * time.Second))
				conn, channels, requests, err := gossh.NewServerConn(raw, config)
				if err != nil {
					return
				}
				defer conn.Close()
				go gossh.DiscardRequests(requests)
				for ch := range channels {
					if handle == nil {
						ch.Reject(gossh.Prohibited, "no channels")
						continue
					}
					go handle(ch)
				}
			}()
		}
	}()
	return listener.Addr().String()
}
