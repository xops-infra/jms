package pui

import (
	"fmt"

	"github.com/elfgzp/ssh"
)

func defaultShow(int, *MenuItem, *ssh.Session, []*MenuItem) bool { return true }

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
	InfoLabel      = fmt.Sprint(`1) AnyQuestions can be asked at https://github.com/xops-infra/jms/issues
2) In the default policy, you are not able to access assets with tag EnvType=prod.
3) Filter supports fuzzy matching of IP/Name/ID.
4) Use ctrl+c to return&flash server.
5) Use ctrl+d to quit.
6) Filter[nu] nu is the server number you can select.
`)
)
