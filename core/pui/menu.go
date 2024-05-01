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
	ServerMenu    *MenuItem
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
-----------------------------------------------------------------------
- 提交bug或者star🌟,👉 https://github.com/xops-infra/jms ;
- 你可以看到所有服务器，[x]表示你没有权限访问，[√]表示你可以访问。
- 2个默认策略:
	1.机器tag:Owner=user;
	2.机器tag:Team=你jms用户信息组一致（通过API管理）
- 过滤支持服务器名称、机器ID、IP地址;
- 进入服务器列表页后使用左右按键翻页，上下按键选择；
- 使用 ctrl+c 取消及刷新机器列表,使用 ctrl+d 退出；
-----------------------------------------------------------------------
请输入关键字，回车进行过滤后选择:
`
)
