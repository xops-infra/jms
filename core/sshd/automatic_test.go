package sshd

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/db"
	"github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
	gossh "golang.org/x/crypto/ssh"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type loginFixture struct {
	mu       sync.Mutex
	allowed  map[string]bool
	attempts []string
	commands int
}

func (f *loginFixture) takeAttempts() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	result := append([]string{}, f.attempts...)
	f.attempts = nil
	return result
}

func loginSetup(t *testing.T) (*loginFixture, model.Server, model.SSHUser) {
	t.Helper()
	log.Default().WithFilename(filepath.Join(t.TempDir(), "login.log")).WithLevel(log.FatalLevel).Init()
	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "login.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&model.Proxy{}, &model.SSHLoginHint{}); err != nil {
		t.Fatal(err)
	}
	previous := app.App
	app.App = &app.Application{Config: &model.Config{WithDB: model.WithPolicy{Enable: true}}, DBIo: db.NewJmsDbService(database)}
	t.Cleanup(func() { app.App = previous })
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := gossh.NewSignerFromKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	f := &loginFixture{allowed: map[string]bool{"ubuntu": true}}
	config := &gossh.ServerConfig{PublicKeyCallback: func(c gossh.ConnMetadata, _ gossh.PublicKey) (*gossh.Permissions, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.attempts = append(f.attempts, c.User())
		if f.allowed[c.User()] {
			return nil, nil
		}
		return nil, errors.New("fixture rejected identity")
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
				raw.SetDeadline(time.Now().Add(10 * time.Second))
				conn, channels, requests, err := gossh.NewServerConn(raw, config)
				if err != nil {
					return
				}
				defer conn.Close()
				go gossh.DiscardRequests(requests)
				for ch := range channels {
					channel, requests, err := ch.Accept()
					if err != nil {
						continue
					}
					go func() {
						defer channel.Close()
						for request := range requests {
							if request.Type == "exec" {
								f.mu.Lock()
								f.commands++
								f.mu.Unlock()
								request.Reply(true, nil)
								channel.SendRequest("exit-status", false, gossh.Marshal(struct{ Status uint32 }{7}))
								return
							}
							request.Reply(false, nil)
						}
					}()
				}
			}()
		}
	}()
	host, portText, _ := net.SplitHostPort(listener.Addr().String())
	var port int
	fmt.Sscan(portText, &port)
	return f, model.Server{ID: "fixture-asset", Host: host, Port: port}, model.SSHUser{UserName: "root", KeyName: "fixture.pem", Base64Pem: base64.StdEncoding.EncodeToString(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: encoded})), Auto: true}
}

func TestAutomaticLoginRemembersSuccessAndExecutesOnlyOnce(t *testing.T) {
	f, server, user := loginSetup(t)
	_, client, actual, err := ConnectSSHClient("caller", server, user)
	if err != nil {
		t.Fatal(err)
	}
	if actual.UserName != "ubuntu" || !reflect.DeepEqual(f.takeAttempts(), []string{"root", "ubuntu"}) {
		t.Fatal("automatic candidates were not tried in order")
	}
	session, err := client.NewSession()
	if err != nil {
		t.Fatal(err)
	}
	if err := session.Run("a command that returns seven"); err == nil {
		t.Fatal("expected remote exit failure")
	}
	session.Close()
	client.Close()
	f.mu.Lock()
	commands := f.commands
	f.mu.Unlock()
	if commands != 1 {
		t.Fatal("command retried")
	}
	_, client, _, err = ConnectSSHClient("caller", server, user)
	if err != nil {
		t.Fatal(err)
	}
	client.Close()
	if !reflect.DeepEqual(f.takeAttempts(), []string{"ubuntu"}) {
		t.Fatal("learned identity was not preferred")
	}
	f.mu.Lock()
	f.allowed = map[string]bool{"ec2-user": true}
	f.mu.Unlock()
	_, client, actual, err = ConnectSSHClient("caller", server, user)
	if err != nil {
		t.Fatal(err)
	}
	client.Close()
	if actual.UserName != "ec2-user" || !reflect.DeepEqual(f.takeAttempts(), []string{"ubuntu", "root", "ec2-user"}) {
		t.Fatal("stale hint was not invalidated or candidates repeated")
	}
}

func TestExplicitOverrideAndDisabledProbeNeverSwitchUsers(t *testing.T) {
	for _, mode := range []string{"explicit", "override", "disabled", "password"} {
		t.Run(mode, func(t *testing.T) {
			f, server, user := loginSetup(t)
			switch mode {
			case "explicit":
				user.Auto = false
			case "override":
				server.User = "root"
			case "disabled":
				disabled := false
				app.App.Config.SSHLogin.AutoUserEnabled = &disabled
			case "password":
				user.Password = "fixture-password"
			}
			_, _, actual, err := ConnectSSHClient("caller", server, user)
			if err == nil || actual.UserName != "root" {
				t.Fatal("strict selection unexpectedly switched identity")
			}
			if mode != "password" && !reflect.DeepEqual(f.takeAttempts(), []string{"root"}) {
				t.Fatal("strict selection retried")
			}
		})
	}
}

func TestAutomaticLoginStopsOnNetworkAndProxyFailures(t *testing.T) {
	_, server, user := loginSetup(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, portText, _ := net.SplitHostPort(listener.Addr().String())
	fmt.Sscan(portText, &server.Port)
	listener.Close()
	_, _, actual, err := ConnectSSHClient("caller", server, user)
	if err == nil || actual.UserName != "root" || targetAuthenticationRejected(err) {
		t.Fatal("network failure triggered fallback")
	}
	proxyAddress := startSSHFixture(t, false, nil)
	proxyHost, proxyPortText, _ := net.SplitHostPort(proxyAddress)
	var proxyPort int
	fmt.Sscan(proxyPortText, &proxyPort)
	proxy := model.Proxy{UUID: "proxy", Name: "fixture", Host: proxyHost, Port: proxyPort, IPPrefix: server.Host, LoginUser: "proxy", LoginPasswd: "fixture-password"}
	if err := app.App.DBIo.DB.Create(&proxy).Error; err != nil {
		t.Fatal(err)
	}
	_, _, actual, err = ConnectSSHClient("caller", server, user)
	if err == nil || !strings.Contains(err.Error(), "stage=proxy_connect") || actual.UserName != "root" || targetAuthenticationRejected(err) {
		t.Fatal("proxy failure triggered fallback")
	}
}

func TestLoginMemoryIsBestEffortAndDoesNotCrossAssets(t *testing.T) {
	f, server, user := loginSetup(t)
	_, client, _, err := ConnectSSHClient("caller", server, user)
	if err != nil {
		t.Fatal(err)
	}
	client.Close()
	f.takeAttempts()
	server.ID = "another-asset"
	_, client, _, err = ConnectSSHClient("caller", server, user)
	if err != nil {
		t.Fatal(err)
	}
	client.Close()
	if !reflect.DeepEqual(f.takeAttempts(), []string{"root", "ubuntu"}) {
		t.Fatal("login hint crossed asset boundary")
	}
	if err := app.App.DBIo.DB.Migrator().DropTable(&model.SSHLoginHint{}); err != nil {
		t.Fatal(err)
	}
	_, client, _, err = ConnectSSHClient("caller", server, user)
	if err != nil {
		t.Fatalf("memory error blocked authentication: %v", err)
	}
	client.Close()
}
