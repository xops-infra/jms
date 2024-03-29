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
	SelectedFunc      func(index int, menuItem *MenuItem, sess *ssh.Session, selectedChain []*MenuItem) (bool, error)
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
	MainLabel           = "Please select,ctrl+c to return,ctrl+d to exit"
	ApproveSummaryLabel = "Please check the approve summary"
	UserLoginLabel      = "Please select ssh user to login"
	NoPermission        = "No permission,Please apply for permission"
	SelectServer        = "Please select server for approve"
	SelectAction        = "Please select action"
	BybLabel            = "\n拜拜👋! 退出时间：%s\n"
	InfoLabel           = `-----------------------------------------------------------------------
欢迎使用jms堡垒机连接工具
1) 问题或者提交BUG，前往 https://github.com/xops-infra/jms/issues；
2) 你可以看到所有服务器，但是在连接或者上传下载时会校验你的权限，如果没有权限可以进行交互申请；
3) 你可以选择不可连接服务器进行权限申请。（2个默认策略：1.tag:Owner=user;2.tag:Team和你jms用户信息组一致）；
4) 过滤支持服务器名称、机器ID、IP地址；
5）进入服务器列表页后使用左右按键翻页，上下按键选择；
6) 使用 ctrl+c 取消及刷新机器列表,使用 ctrl+d 退出；
7) Filter[nu] 方括号内数量表示你能访问的机器总数；
-----------------------------------------------------------------------
请输入关键字，回车进行过滤后选择:

`
)
