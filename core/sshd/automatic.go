package sshd

import (
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/xops-infra/jms/app"
	. "github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
	gossh "golang.org/x/crypto/ssh"
)

// ConnectionError identifies the hop owning an error without inspecting a proxy's error text.
type ConnectionError struct {
	Stage, Detail string
	Err           error
}

func (e *ConnectionError) Error() string {
	return fmt.Sprintf("SSH connection failed (stage=%s %s): %v", e.Stage, e.Detail, e.Err)
}
func (e *ConnectionError) Unwrap() error { return e.Err }

func targetAuthenticationRejected(err error) bool {
	var connection *ConnectionError
	return errors.As(err, &connection) && connection.Stage == "target_connect" &&
		strings.Contains(connection.Err.Error(), "ssh: unable to authenticate, attempted methods")
}

func automaticUsersEnabled() bool {
	return app.App == nil || app.App.Config == nil || app.App.Config.SSHLogin.AutoUserEnabled == nil || *app.App.Config.SSHLogin.AutoUserEnabled
}

// NewSSHClient retains the explicit-identity API used by file transfers and other callers.
func NewSSHClient(user string, server Server, selected SSHUser) (*gossh.Client, *gossh.Client, error) {
	proxy, client, _, err := ConnectSSHClient(user, server, selected)
	return proxy, client, err
}

// ConnectSSHClient returns the actual identity, allowing callers to audit it accurately.
// Only handshakes are retried: callers create and execute their session exactly once.
func ConnectSSHClient(user string, server Server, selected SSHUser) (*gossh.Client, *gossh.Client, SSHUser, error) {
	automatic := selected.Auto && automaticUsersEnabled() && selected.Password == "" && strings.TrimSpace(server.User) == ""
	address := net.JoinHostPort(server.Host, fmt.Sprint(server.Port))
	fingerprint := ""
	if selected.Password == "" && (selected.Base64Pem != "" || selected.KeyName != "") {
		var signer gossh.Signer
		var err error
		if selected.Base64Pem != "" {
			signer, err = getSignerFromBase64(selected.Base64Pem)
		} else {
			signer, err = getSignerFromLocal(app.App.SSHDir + strings.TrimPrefix(selected.KeyName, "/"))
		}
		if err != nil {
			return nil, nil, selected, fmt.Errorf("SSH target credentials failed (stage=target_credentials target=%q user=%q key=%q): %w", address, selected.UserName, selected.KeyName, err)
		}
		fingerprint = gossh.FingerprintSHA256(signer.PublicKey())
	}
	memory := app.App != nil && app.App.DBIo != nil && app.App.DBIo.DB != nil && server.ID != "" && fingerprint != ""
	remembered := ""
	if automatic && memory {
		var err error
		remembered, err = app.App.DBIo.SSHLoginUsername(server.ID, address, fingerprint)
		if err != nil {
			log.Warnf("SSH login memory read failed (asset=%q): %v", server.ID, err)
		}
	}
	candidates := []string{selected.UserName}
	if automatic {
		candidates = []string{remembered, selected.UserName, "root", "ubuntu", "ec2-user"}
	}
	if override := strings.TrimSpace(server.User); selected.Auto && override != "" {
		candidates = []string{override}
	}
	seen := make(map[string]bool)
	var last error
	for _, username := range candidates {
		username = strings.TrimSpace(username)
		if username == "" || seen[username] {
			continue
		}
		seen[username] = true
		selected.UserName, selected.Auto = username, false
		log.Infof("SSH authentication attempt asset=%q target=%q user=%q key=%q automatic=%t", server.ID, address, username, selected.KeyName, automatic)
		proxy, client, err := newSSHClient(user, server, selected)
		if err == nil {
			if memory {
				if err := app.App.DBIo.RememberSSHLogin(server.ID, address, fingerprint, username); err != nil {
					log.Warnf("SSH login memory write failed (asset=%q): %v", server.ID, err)
				}
			}
			log.Infof("SSH authenticated asset=%q target=%q user=%q key=%q", server.ID, address, username, selected.KeyName)
			return proxy, client, selected, nil
		}
		last = err
		if !automatic || !targetAuthenticationRejected(err) {
			return nil, nil, selected, err
		}
		log.Infof("SSH target rejected identity asset=%q user=%q; checking next automatic candidate", server.ID, username)
		if memory && username == remembered {
			if err := app.App.DBIo.ForgetSSHLogin(server.ID, address, fingerprint, username); err != nil {
				log.Warnf("SSH login memory invalidation failed (asset=%q): %v", server.ID, err)
			}
		}
	}
	if last == nil {
		last = errors.New("SSH login username is empty")
	}
	return nil, nil, selected, last
}
