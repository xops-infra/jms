package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/xops-infra/noop/log"
	gossh "golang.org/x/crypto/ssh"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/sshd"
	"github.com/xops-infra/jms/model"
)

type wsMessage struct {
	Type      string `json:"type"`
	Data      string `json:"data"`
	Cols      int    `json:"cols"`
	Rows      int    `json:"rows"`
	SessionID string `json:"session_id"`
}

var terminalUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type wsWriter struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (w *wsWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	msg := wsMessage{Type: "data", Data: string(p)}
	if err := w.conn.WriteJSON(msg); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (w *wsWriter) Send(msg wsMessage) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteJSON(msg)
}

// terminalWS handles websocket terminal sessions
func terminalWS(c *gin.Context) {
	v, ok := c.Get("auth_user")
	if !ok {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}
	user := v.(model.User)
	if user.Username == nil {
		c.String(http.StatusUnauthorized, "unauthorized")
		return
	}

	host := c.Query("host")
	if host == "" {
		c.String(http.StatusBadRequest, "host required")
		return
	}

	sshUserQuery := c.Query("user")
	sshKeyQuery := c.Query("key")
	sessionID := c.Query("session_id")
	cols := parseIntDefault(c.Query("cols"), 120)
	rows := parseIntDefault(c.Query("rows"), 32)

	server, sshUsers, err := app.App.Sshd.SshdIO.GetSSHUsersByHostLive(host)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}

	matchPolicies := app.App.Sshd.SshdIO.GetUserPolicys(*user.Username)
	if !app.App.Sshd.SshdIO.MatchPolicy(user, model.Connect, *server, matchPolicies, false) {
		c.String(http.StatusForbidden, "no permission")
		return
	}

	sshUser, err := selectSSHUser(sshUsers, sshUserQuery, sshKeyQuery)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}

	_, upstream, err := sshd.NewSSHClient(*user.Username, *server, sshUser)
	if err != nil {
		c.String(http.StatusBadRequest, err.Error())
		return
	}
	defer upstream.Close()

	sess, err := upstream.NewSession()
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	defer sess.Close()

	if err := sess.RequestPty("xterm-256color", rows, cols, gossh.TerminalModes{}); err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	conn, err := terminalUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	writer := &wsWriter{conn: conn}
	var outWriter io.Writer = writer
	var auditFile io.Closer
	if app.App.Config.WithVideo.Enable {
		logFile, err := sshd.NewAuditLog(*user.Username, server.Host)
		if err != nil {
			log.Warnf("create audit log failed: %v", err)
		} else {
			auditFile = logFile
			outWriter = io.MultiWriter(writer, logFile)
		}
	}
	if auditFile != nil {
		defer auditFile.Close()
	}

	sess.Stdout = outWriter
	sess.Stderr = outWriter
	stdin, err := sess.StdinPipe()
	if err != nil {
		_ = writer.Send(wsMessage{Type: "exit", Data: err.Error()})
		return
	}

	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	_ = writer.Send(wsMessage{Type: "session", SessionID: sessionID})

	tmuxEnabled := app.App.Config.Terminal.TmuxEnable == nil || *app.App.Config.Terminal.TmuxEnable
	tmuxAvailable := false
	if tmuxEnabled {
		tmuxAvailable = isTmuxAvailable(upstream)
		if !tmuxAvailable {
			log.Infof("tmux not found on %s, fallback to shell", server.Host)
		}
	}

	if tmuxEnabled && tmuxAvailable {
		cmd := fmt.Sprintf("tmux new -A -s %s", sessionID)
		if err := sess.Start(cmd); err != nil {
			log.Warnf("tmux start failed: %v", err)
			if err := sess.Shell(); err != nil {
				_ = writer.Send(wsMessage{Type: "exit", Data: err.Error()})
				return
			}
		}
	} else {
		if err := sess.Shell(); err != nil {
			_ = writer.Send(wsMessage{Type: "exit", Data: err.Error()})
			return
		}
	}

	errCh := make(chan error, 2)

	go func() {
		defer close(errCh)
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			var m wsMessage
			if err := json.Unmarshal(msg, &m); err != nil {
				continue
			}
			switch m.Type {
			case "input":
				if m.Data != "" {
					if _, err := stdin.Write([]byte(m.Data)); err != nil {
						errCh <- err
						return
					}
				}
			case "resize":
				if m.Rows > 0 && m.Cols > 0 {
					_ = sess.WindowChange(m.Rows, m.Cols)
				}
			case "ping":
				_ = writer.Send(wsMessage{Type: "pong"})
			}
		}
	}()

	waitErr := sess.Wait()
	if waitErr != nil {
		log.Warnf("terminal session ended: %v", waitErr)
	}
	_ = writer.Send(wsMessage{Type: "exit"})
	select {
	case <-errCh:
	default:
	}
}

func parseIntDefault(v string, d int) int {
	if v == "" {
		return d
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return d
	}
	return i
}

func selectSSHUser(users []model.SSHUser, preferUser, preferKey string) (model.SSHUser, error) {
	if len(users) == 0 {
		return model.SSHUser{}, fmt.Errorf("no ssh users")
	}
	if preferUser != "" || preferKey != "" {
		for _, u := range users {
			if preferUser != "" && u.UserName != preferUser {
				continue
			}
			if preferKey != "" && u.KeyName != preferKey {
				continue
			}
			return u, nil
		}
		if preferUser != "" && preferKey != "" {
			return model.SSHUser{}, fmt.Errorf("ssh user %s with key %s not found", preferUser, preferKey)
		}
		if preferUser != "" {
			return model.SSHUser{}, fmt.Errorf("ssh user %s not found", preferUser)
		}
		if preferKey != "" {
			return model.SSHUser{}, fmt.Errorf("ssh key %s not found", preferKey)
		}
	}
	return users[0], nil
}

func isTmuxAvailable(client *gossh.Client) bool {
	sess, err := client.NewSession()
	if err != nil {
		log.Warnf("tmux probe session failed: %v", err)
		return false
	}
	defer sess.Close()
	if err := sess.Run("command -v tmux >/dev/null 2>&1"); err != nil {
		return false
	}
	return true
}
