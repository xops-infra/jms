package dingtalk

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	dt "github.com/xops-infra/go-dingtalk-sdk-wrapper"
	"github.com/xops-infra/noop/log"

	"github.com/xops-infra/jms/app"
	. "github.com/xops-infra/jms/model"
)

var robot *dt.RobotClient

func init() {
	robot = dt.NewRobotClient()
}

func CreateApproval(applicant string, values []dt.FormComponentValue) (string, error) {

	user, err := app.App.DBIo.DescribeUser(applicant)
	if err != nil {
		return "", err
	}
	log.Debugf(tea.Prettify(user), tea.Prettify(values))
	if user.DingtalkID == nil || user.DingtalkDeptID == nil {
		return "", fmt.Errorf("get user %s dingtalkid or dingtalkdeptid failed", applicant)
	}

	// 主动刷新token
	err = app.App.Schedule.DingTalkClient.SetAccessToken()
	if err != nil {
		log.Errorf("SetAccessToken failed! %s", err)
		return "", err
	}
	return app.App.Schedule.DingTalkClient.Workflow.CreateProcessInstance(&dt.CreateProcessInstanceInput{
		ProcessCode:         app.App.Config.WithDingtalk.ProcessCode,
		OriginatorUserID:    tea.StringValue(user.DingtalkID),
		DeptId:              tea.StringValue(user.DingtalkDeptID),
		FormComponentValues: values,
	}, app.App.Schedule.DingTalkClient.AccessToken.Token)
}

// 同步钉钉用户到数据库user表
func LoadUsers() error {
	err := app.App.Schedule.DingTalkClient.SetAccessToken()
	if err != nil {
		log.Errorf("SetAccessToken failed! %s", err)
	}
	departIDChan := make(chan int64, 2000)
	go getDepart(departIDChan)
	startTime := time.Now()
	log.Infof("load dingtalk users start")
	for depart := range departIDChan {
		log.Infof("load depart %d", depart)
		departRes, err := app.App.Schedule.DingTalkClient.Depart.GetDepartmentIDs(&dt.GetDepartmentsIDInput{
			DeptID: depart,
		}, app.App.Schedule.DingTalkClient.AccessToken.Token)
		if err != nil {
			return err
		}
		for _, v := range departRes {
			departIDChan <- v
		}
		_users, err := app.App.Schedule.DingTalkClient.User.GetUsers(&dt.GetUsersInput{
			DeptID: depart,
			Size:   100,
			Cursor: 0,
		}, app.App.Schedule.DingTalkClient.AccessToken.Token)
		if err != nil {
			return err
		}
		err = saveDingtalkUsers(_users)
		if err != nil {
			return err
		}
	}

	log.Infof("load dingtalk users success, cost %v", time.Since(startTime))
	return nil
}

func getDepart(c chan int64) {
	input := &dt.GetDepartmentsIDInput{
		DeptID: int64(1),
	}
	departRes, err := app.App.Schedule.DingTalkClient.Depart.GetDepartmentIDs(input, app.App.Schedule.DingTalkClient.AccessToken.Token)
	if err != nil {
		panic(err)
	}
	for _, v := range departRes {
		c <- v
	}
	for {
		time.Sleep(1 * time.Second)
		if len(c) == 0 {
			close(c)
			break
		}
	}
}

func saveDingtalkUsers(users []*dt.UserInfo) error {
	for _, user := range users {
		u, err := app.App.DBIo.DescribeUser(strings.Split(user.Email, "@")[0])
		if err != nil {
			if strings.Contains(err.Error(), "record not found") {
				// create
				_, err = app.App.DBIo.CreateUser(&UserRequest{
					Username:       tea.String(strings.Split(user.Email, "@")[0]),
					Email:          tea.String(user.Email),
					Passwd:         tea.String(user.Email),
					DingtalkDeptID: tea.String(strconv.FormatInt(user.DeptIDList[0], 10)),
					DingtalkID:     tea.String(user.UserID),
				})
				if err != nil {
					return fmt.Errorf("create dingtalk user failed! %s", err)
				}
				log.Infof("save dingtalk user %s", user.Email)
			}
			return err
		}
		// update
		err = app.App.DBIo.UpdateUser(u.ID, UserRequest{
			Username:       tea.String(strings.Split(user.Email, "@")[0]),
			Email:          tea.String(user.Email),
			DingtalkDeptID: tea.String(strconv.FormatInt(user.DeptIDList[0], 10)),
			DingtalkID:     tea.String(user.UserID),
		})
		if err != nil {
			return fmt.Errorf("update dingtalk user failed! %s", err)
		}
		log.Infof("update dingtalk user %s", user.Email)
	}
	return nil
}

// 本地审批列表到云上获取审批状态并更新。
func LoadApproval() {
	timeStart := time.Now()
	var successes []string
	err := app.App.Schedule.DingTalkClient.SetAccessToken() // 更新 token
	if err != nil {
		log.Errorf("SetAccessToken failed! %s", err)
		return
	}
	log.Infof("tonkens: %s", app.App.Schedule.DingTalkClient.AccessToken.Token)
	// 获取审批列表
	policies, err := app.App.DBIo.QueryAllPolicy()
	if err != nil {
		log.Errorf("QueryAllPolicy failed! %s", err)
		return
	}
	for _, policy := range policies {
		if policy.ApprovalID == "" {
			continue
		}
		if strings.Contains(policy.Approver, "BusinessId") {
			// 已经更新过的审批不再更新
			continue
		}
		resp, err := app.App.Schedule.DingTalkClient.Workflow.GetProcessInstance(policy.ApprovalID, app.App.Schedule.DingTalkClient.AccessToken.Token)
		if err != nil {
			log.Errorf("GetProcessInstance failed! %s", err)
		}
		// 更新
		if !resp.Success {
			continue
		}
		if resp.Result.Result != nil {
			update := ApprovalResult{
				Applicant: tea.String(fmt.Sprintf("%s: %s", "BusinessId", *resp.Result.BusinessId)),
				IsPass:    tea.Bool(false),
			}
			switch *resp.Result.Status {
			case "COMPLETED":
				if *resp.Result.Result == "agree" {
					update.IsPass = tea.Bool(true)
				}
			case "TERMINATED":
				update.Applicant = tea.String("BusinessId: TERMINATED")
				update.IsPass = tea.Bool(false)
			default:
				continue
			}
			err = app.App.DBIo.UpdatePolicyStatus(policy.ID, update)
			if err != nil {
				log.Errorf("UpdatePolicyStatus failed! %s", err)
				continue
			}
			successes = append(successes, *resp.Result.BusinessId)
			log.Infof("update dingtalk approval %s(%s) to %v from DTCloud", *resp.Result.BusinessId, *resp.Result.Title, *update.IsPass)
		}
	}
	log.Infof("load dingtalk approval success, %d/%d, cost %v", len(successes), len(policies), time.Since(timeStart))
}

// 发送钉钉机器人通知
func SendRobotText(robotToken, content, userID string) error {
	if robotToken == "" {
		return fmt.Errorf("robot token is empty, can not send robot message")
	}
	// 发送通知
	err := robot.SendMessage(context.Background(), &dt.SendMessageRequest{
		AccessToken: robotToken,
		MessageContent: dt.MessageContent{
			MsgType: "text",
			Text: dt.TextBody{
				Content: content,
			},
			At: dt.AtBody{
				AtUserIDS: []string{userID},
			},
		},
	})
	if err != nil {
		if strings.Contains(err.Error(), "ok") {
			return nil
		}
	}
	return err
}
