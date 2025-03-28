package pui

import (
	"github.com/elfgzp/ssh"
)

// MenuItem menu item
type MenuItem struct {
	Label             string
	Info              map[string]string
	IsShow            func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) bool
	SubMenuTitle      string
	GetSubMenu        func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) []MenuItem
	SelectedFunc      func(index int, menuItem MenuItem, sess *ssh.Session, selectedChain []MenuItem) (bool, error)
	NoSubMenuInfo     string
	BackAfterSelected bool
	BackOptionLabel   string
}

// MainMenu main menu
var (
	// ServerMenu    MenuItem
	serverInfoKey = "serverKey"
	serverHost    = "serverHost"
	serverUser    = "serverUser"
)

var (
	MainLabel           = "Please select,ctrl+c to return,ctrl+d to exit"
	ApproveSummaryLabel = "Please check the approve summary"
	UserLoginLabel      = "Please select ssh user to login"
	NoPermission        = "No permission,Please apply for permission"
	SelectServer        = "Please select server for approve"
	SelectAction        = "Please select action"
	BybLabel            = "\n拜拜! 退出时间：%s\n"
	InfoLabel           = `-----------------------------------------------------------------------
欢迎使用jms堡垒机连接工具 版本: %s %s
项目地址: https://github.com/xops-infra/jms
-----------------------------------------------------------------------
[快捷键说明]
- 主菜单: Ctrl+C 刷新列表, Ctrl+D 退出程序
- 子菜单: Ctrl+C 返回上层, Ctrl+D 退出程序
- 服务器: ↑↓ 选择, ←→ 翻页, Enter 确认

[服务器说明]
- [√] 有权限访问
- [x] 无权限访问（需申请权限）
- 支持过滤：服务器名称/ID/IP
-----------------------------------------------------------------------
`
)
