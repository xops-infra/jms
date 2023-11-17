package jump

import (
	"github.com/elfgzp/ssh"
	"github.com/xops-infra/jms/core/pui"
	"github.com/xops-infra/jms/core/sshd"
	gossh "golang.org/x/crypto/ssh"
)

// Service Service
type Service struct {
	sess      *ssh.Session
	persionUI *pui.PUI
}

func (jps *Service) setSession(sess *ssh.Session) {
	jps.sess = sess
}

// Run jump
func (jps *Service) Run(sess *ssh.Session) {
	defer func() {
		(*sess).Exit(0)
	}()

	if false {
		sshd.Info("Please login again with your new acount. \n", sess)
		sshConn := (*sess).Context().Value(ssh.ContextKeyConn).(gossh.Conn)
		sshConn.Close()
		return
	}
	jps.setSession(sess)
	jps.persionUI = &pui.PUI{}
	jps.persionUI.SetSession(jps.sess)
	jps.persionUI.ShowMainMenu()
}
