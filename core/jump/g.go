package jump

import (
	"fmt"
	"time"

	"github.com/elfgzp/ssh"

	"github.com/xops-infra/jms/core/pui"
	"github.com/xops-infra/jms/io"
	"github.com/xops-infra/noop/log"
)

// Service Service
type Service struct {
	persionUI *pui.PUI
}

func NewSession(p *io.PolicyIO, sess *ssh.Session, timeout time.Duration) *Service {
	return &Service{
		persionUI: pui.NewPui(p, sess, timeout),
	}
}

// Run jump
func (jps *Service) Run() {
	// 设置超时退出
	go func() {
		for {
			// 用户主动 退出的也要自己终端
			if jps.persionUI.IsLogout() {
				log.Debugf("exit by user logout")
				break
			}
			time.Sleep(1 * time.Second)
			if jps.persionUI.IsTimeout() {
				isExit := false
				// 10 秒倒计时，如果捕捉到输入则取消退出，刷新超时时间
				for i := 15; i > 0; i-- {
					time.Sleep(1 * time.Second)
					if !jps.persionUI.IsTimeout() {
						isExit = true
						break
					}
					jps.persionUI.SessionWrite(fmt.Sprintf("\033[2K\r系统超时设置：%s。%2.d秒后自动退出。ctrl+c 取消退出！", jps.persionUI.GetTimeout(), i))
				}
				if !isExit {
					jps.persionUI.Exit()
					log.Debugf("exit by timeout")
					break
				}
			}
		}
	}()
	jps.persionUI.ShowMainMenu()
}
