package pui

import (
	"github.com/elfgzp/ssh"
)

// MenuItem menu item
type MenuItem struct {
	Label             string
	Info              map[string]string
	IsShow            func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) bool
	SubMenuTitle      string
	GetSubMenu        func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) []*MenuItem
	SelectedFunc      func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) error
	NoSubMenuInfo     string
	BackAfterSelected bool
	BackOptionLabel   string
}

// MainMenu main menu
var (
	ServerMenu                 *MenuItem
	serverInfoKey              = "serverKey"
	serverHost                 = "serverHost"
	serverUser                 = "serverUser"
	serverProxy                = "serverProxy"
	serverProxyUser            = "serverProxyUser"
	serverProxyKeyIdentityFile = "serverProxyKeyIdentityFile"
)

var (
	MainLabel      = "Please select,ctrl+c to return,ctrl+d to exit"
	UserLoginLabel = "Please select ssh user to login"
	InfoLabel      = `-----------------------------------------------------------------------
欢迎使用jms堡垒机连接工具
1) 问题或者提交BUG，前往 https://github.com/xops-infra/jms/issues；
2) 默认策略下你讲不能访问机器标签EnvType=prod的机器，Admin组用户除外；
3) 过滤支持服务器名称、机器ID、IP地址；
4) 使用 ctrl+c 取消及刷新机器列表；
5）进入服务器列表页后使用左右按键翻页，上下按键选择；
6) 使用 ctrl+d 退出；
7) Filter[nu] 方括号内数量表示你能访问的机器总数；
-----------------------------------------------------------------------
请输入关键字，回车进行过滤后选择:

`
)
