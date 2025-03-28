package pui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/elfgzp/ssh"
	"github.com/manifoldco/promptui"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/sshd"
	. "github.com/xops-infra/jms/model"
)

// PUI pui
type PUI struct {
	sess             *ssh.Session
	timeOut          time.Duration
	stopCheckTimeout bool      // 当用户进入服务器后暂停检测
	lastActive       time.Time // 最后活跃时间
	isLogout         bool      // 主动退出的标记
	menuItem         []MenuItem
}

func NewPui(s *ssh.Session, timeout time.Duration) *PUI {
	return &PUI{
		sess:             s,
		timeOut:          timeout,
		lastActive:       time.Now(),
		isLogout:         false,
		stopCheckTimeout: false,
	}
}

func (ui *PUI) sessionWrite(msg string) error {
	_, err := (*ui.sess).Write([]byte(msg))
	return err
}

// pause
func (ui *PUI) pause() {
	ui.stopCheckTimeout = true
}

// resume
func (ui *PUI) resume() {
	log.Warnf("resume timeout check")
	ui.stopCheckTimeout = false
}

// exit
func (ui *PUI) exit() {
	ui.sessionWrite(fmt.Sprintf(BybLabel, time.Now().Local().Format("2006-01-02 15:04:05")))
	ui.isLogout = true
	(*ui.sess).Close() // 只关闭当前会话
}

// 当用户连接主机的时候这个判断永远不超时
func (ui *PUI) isTimeout() bool {
	if ui.stopCheckTimeout {
		return false
	}
	log.Debugf("lastActive: %v, timeOut: %v", ui.lastActive, ui.timeOut)
	return time.Since(ui.lastActive) > ui.timeOut
}

// getTimeout
func (ui *PUI) getTimeout() string {
	return fmt.Sprint(ui.timeOut)
}

// get username
func (ui *PUI) getUsername() string {
	return (*ui.sess).User()
}

// flashTimeout flash timeout
func (ui *PUI) flashTimeout() {
	ui.lastActive = time.Now()
}

// showMenu show menu
func (ui *PUI) showMenu(label string, menu []MenuItem, BackOptionLabel string, selectedChain []MenuItem) {
loopMenu:
	for {
		menuLabels := make([]string, 0) // 菜单，用于显示
		menuItems := make([]MenuItem, 0)
		if ui.menuItem == nil {
			log.Debugf("menu is nil, label: %s", label)
		}

		// 返回顶级菜单
		log.Debugf("label: %s MainLabel:%s", label, MainLabel)
		switch label {
		case MainLabel:
			// 顶级菜单，如果有审批则主页支持选择审批或者服务器
			// menu = make([]MenuItem, 0)

			if app.App.Config.WithDB.Enable && !app.App.Config.WithDingtalk.Enable {
				// 没有审批策略时候，会在 admin 服务器选择列表里面显示审批菜单
				policies, err := app.App.DBIo.NeedApprove((*ui.sess).User())
				if err != nil {
					log.Errorf("Get need approve policy for admin error: %s", err)
				}
				if len(policies) > 0 {
					sshd.Info(fmt.Sprintf("作为管理员，有新的审批工单(%d)待处理。", len(policies)), ui.sess)
					ui.menuItem = append(ui.menuItem, getApproveMenu(policies)...)
				}
			}

			_menus, err := ui.getServersMenuV2(ui.sess)
			if err != nil {
				sshd.ErrorInfo(err, ui.sess)
				break loopMenu
			}
			{
				// 实现新旧菜单内容的合并
				newMenus := make([]MenuItem, 0)
				newMenus = append(newMenus, _menus...)
				ui.menuItem = newMenus
			}
			filter, err := ui.inputFilter(app.GetBroadcast())
			if err != nil {
				if strings.Contains(err.Error(), "exit") {
					return
				}
				sshd.ErrorInfo(err, ui.sess)
				break loopMenu
			}
			if filter == "^C" {
				continue // 在主菜单，^C 刷新当前菜单
			}
			for index, menuItem := range ui.menuItem {
				log.Debugf("menu: %s", tea.Prettify(menuItem))
				if menuItem.IsShow == nil || menuItem.IsShow(index, menuItem, ui.sess, selectedChain) {
					if !strings.Contains(menuItem.Label, filter) {
						continue
					}
					menuLabels = append(menuLabels, menuItem.Label)
					menuItems = append(menuItems, menuItem)
				}
			}
		default:
			for index, menuItem := range menu {
				if menuItem.IsShow == nil || menuItem.IsShow(index, menuItem, ui.sess, selectedChain) {
					menuLabels = append(menuLabels, menuItem.Label)
					menuItems = append(menuItems, menuItem)
				}
			}
		}

		log.Debugf("menuLabels: %s", tea.Prettify(menuLabels))
		if len(menuLabels) == 0 {
			continue
		}
		menuLabels = append(menuLabels, BackOptionLabel) // 添加返回选项
		backIndex := len(menuLabels) - 1                 // 返回选项的索引
		menuPui := promptui.Select{
			Label:  label,
			Size:   15, // 菜单栏服务器数最大为15
			Items:  menuLabels,
			Stdin:  *ui.sess,
			Stdout: *ui.sess,
		}

		// get sub menu label
		index, subMenuLabel, err := menuPui.Run()
		if err != nil {
			// ^C ^D is not error
			if strings.Contains(err.Error(), "^C") {
				log.Debugf(label, MainLabel)
				if label == MainLabel {
					// 在主菜单，^C 刷新当前菜单
					continue
				}
				// 在子菜单，^C 返回上一层
				break loopMenu
			} else if strings.Contains(err.Error(), "^D") {
				ui.exit()
				return
			} else {
				log.Errorf("Select menu error %s\n", err)
			}
			continue
		}
		if index == backIndex {
			// 返回上一级菜单，如果主菜单了则退无可退。
			if label == MainLabel {
				log.Debugf("main menu, no back")
				continue
			} else {
				break
			}
		}

		// get sub menu
		selected := menuItems[index]
		if selected.GetSubMenu != nil {
			getSubMenu := selected.GetSubMenu
			subMenu := getSubMenu(index, selected, ui.sess, selectedChain)

			if len(subMenu) > 0 {
				back := "back"
				if selected.BackOptionLabel != "" {
					back = selected.BackOptionLabel
				}
				if selected.SubMenuTitle != "" {
					subMenuLabel = selected.SubMenuTitle
				}
				ui.showMenu(subMenuLabel, subMenu, back, append(selectedChain, selected))
			} else {
				noSubMenuInfo := "No options under this menu ..."
				if selected.NoSubMenuInfo != "" {
					noSubMenuInfo = selected.NoSubMenuInfo
				}
				sshd.ErrorInfo(errors.New(noSubMenuInfo), ui.sess)
			}
		}

		// run selected func
		if selected.SelectedFunc != nil {
			selectedFunc := selected.SelectedFunc
			isTop, err := selectedFunc(index, selected, ui.sess, selectedChain)
			if err != nil {
				sshd.ErrorInfo(err, ui.sess)
			}
			if isTop {
				label = MainLabel
			}
		}
	}
	log.Debugf("pui exit")
}

// inputFilter input filter
func (ui *PUI) inputFilter(broadcast *Broadcast) (string, error) {
	ui.flashTimeout()
	defer ui.sessionWrite("\033c") // clear
	// 发送屏幕清理指令
	// 发送当前时间
	ui.sessionWrite(fmt.Sprintf("Last connect time: %s\t OnLineUser: %d\t AllServerCount: %d\n",
		time.Now().Local().Format("2006-01-02 15:04:05"), app.App.Sshd.UserCache.ItemCount(), app.App.DBIo.GetServerCount(),
	))
	// 发送欢迎信息
	if broadcast != nil {
		ui.sessionWrite(fmt.Sprintf(InfoLabel, app.App.Version, "\n"+broadcast.Message))
	} else {
		ui.sessionWrite(fmt.Sprintf(InfoLabel, app.App.Version, ""))
	}
	prompt := promptui.Prompt{
		Label:  "请输入关键字，回车进行过滤后选择",
		Stdin:  *ui.sess,
		Stdout: *ui.sess,
	}
	filter, err := prompt.Run()
	if err != nil {
		// ^C ^D is not error
		if err.Error() == "^C" {
			return "^C", nil
		} else if err.Error() == "^D" {
			ui.exit()
			return "", fmt.Errorf("exit")
		}
		log.Errorf("Prompt error: %s", err)
		return "", err
	}
	log.Debugf("Filter: %s", filter)
	return filter, nil
}

// ShowMainMenu show main menu
func (ui *PUI) ShowMainMenu() {
	go ui.timeoutCheck()
	MainMenu := make([]MenuItem, 0)
	selectedChain := make([]MenuItem, 0)
	ui.showMenu(MainLabel, MainMenu, "Quit", selectedChain)
}

func (ui *PUI) timeoutCheck() {
	for {
		// 用户主动 退出的也要直接中断
		if ui.isLogout {
			log.Debugf("%s exit by user logout", ui.getUsername())
			break
		}
		time.Sleep(1 * time.Second)
		log.Debugf("system timeout check for %s", ui.getUsername())
		if ui.isTimeout() {
			isExit := false
			// 10 秒倒计时，如果捕捉到输入则取消退出，刷新超时时间
			for i := 15; i > 0; i-- {
				time.Sleep(1 * time.Second)
				if !ui.isTimeout() {
					isExit = true
					break
				}
				ui.sessionWrite(fmt.Sprintf("\033[2K\r系统超时设置：%s。%2.d秒后自动退出。ctrl+c 取消退出！", ui.getTimeout(), i))
			}
			if !isExit {
				ui.exit()
				log.Debugf("exit by timeout")
				break
			}
		}
	}
}
