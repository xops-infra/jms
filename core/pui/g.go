package pui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/elfgzp/ssh"
	"github.com/manifoldco/promptui"
	"github.com/patsnapops/noop/log"
	"github.com/xops-infra/jms/core/sshd"
)

// PUI pui
type PUI struct {
	sess *ssh.Session
}

// SetSession SetSession
func (ui *PUI) SetSession(s *ssh.Session) {
	ui.sess = s
}

// ShowMenu show menu
func (ui *PUI) ShowMenu(label string, menu []*MenuItem, BackOptionLabel string, selectedChain []*MenuItem) {
	for {
		menuLabels := make([]string, 0)
		menuItems := make([]*MenuItem, 0)

		if menu == nil {
			break
		}

		if strings.HasPrefix(label, MainLabel) {
			menu = make([]*MenuItem, 0)
			menu = append(menu, GetServersMenuV2((*ui.sess).User())...)
			filter, err := ui.inputFilter(len(menu))
			if err != nil {
				continue
			}
			for index, menuItem := range menu {
				if menuItem.IsShow == nil || menuItem.IsShow(index, menuItem, ui.sess, selectedChain) {
					log.Debugf("index: %d label: %s", index, menuItem.Label)
					if !strings.Contains(menuItem.Label, filter) {
						continue
					}
					menuLabels = append(menuLabels, menuItem.Label)
					menuItems = append(menuItems, menuItem)
				}

			}

		} else {
			for index, menuItem := range menu {
				if menuItem.IsShow == nil || menuItem.IsShow(index, menuItem, ui.sess, selectedChain) {
					log.Debugf("index: %d label: %s", index, menuItem.Label)
					menuLabels = append(menuLabels, menuItem.Label)
					menuItems = append(menuItems, menuItem)
				}
			}
		}

		if len(menuLabels) == 0 {
			continue
		}
		menuLabels = append(menuLabels, BackOptionLabel)
		backIndex := len(menuLabels) - 1
		menuPui := promptui.Select{
			Label:  label,
			Size:   15,
			Items:  menuLabels,
			Stdin:  *ui.sess,
			Stdout: *ui.sess,
		}

		index, subMenuLabel, err := menuPui.Run()

		log.Debugf("Selected index: %d subMenuLabel: %+v err: %v", index, subMenuLabel, err)
		if err != nil {
			// ^C ^D is not error
			if err.Error() == "^C" {
				if strings.HasPrefix(label, MainLabel) {
					continue
				} else {
					break
				}

			} else if err.Error() == "^D" {
				(*ui.sess).Close()
				break
			}
			log.Errorf("Select menu error %s\n", err)
			break
		}

		if index == backIndex {
			break
		}

		selected := menuItems[index]

		log.Debugf("Selected: %+v", tea.Prettify(selected.Info))

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
				ui.ShowMenu(subMenuLabel, subMenu, back, append(selectedChain, selected))
			} else {
				noSubMenuInfo := "No options under this menu ... "
				if selected.NoSubMenuInfo != "" {
					noSubMenuInfo = selected.NoSubMenuInfo
				}
				sshd.ErrorInfo(errors.New(noSubMenuInfo), ui.sess)
			}
		}

		if selected.SelectedFunc != nil {
			selectedFunc := selected.SelectedFunc
			log.Debugf("Run selectFunc %+v", selectedFunc)
			err := selectedFunc(index, selected, ui.sess, selectedChain)
			if err != nil {
				log.Errorf("Run selected func err: %s", err)
				sshd.ErrorInfo(err, ui.sess)
			}
			if selected.BackAfterSelected == true {
				break
			}
		}
	}
}

// inputFilter input filter
func (ui *PUI) inputFilter(nu int) (string, error) {
	// write InfoLabel
	(*ui.sess).Write([]byte(InfoLabel))
	prompt := promptui.Prompt{
		Label:  fmt.Sprintf("Filter[%d]", nu),
		Stdin:  *ui.sess,
		Stdout: *ui.sess,
	}
	filter, err := prompt.Run()
	if err != nil {
		// ^C ^D is not error
		if err.Error() == "^C" {
			return "", err
		} else if err.Error() == "^D" {
			(*ui.sess).Close()
			return "", nil
		}
		log.Errorf("Prompt error: %s", err)
		return "", err
	}
	log.Debugf("Filter: %s", filter)
	return filter, nil
}

// ShowMainMenu show main menu
func (ui *PUI) ShowMainMenu() {
	MainMenu := make([]*MenuItem, 0)
	selectedChain := make([]*MenuItem, 0)
	ui.ShowMenu(MainLabel, MainMenu, "Quit", selectedChain)
}
