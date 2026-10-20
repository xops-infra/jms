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
	err = app.App.Scheduler.DingTalkClient.SetAccessToken()
	if err != nil {
		log.Errorf("SetAccessToken failed! %s", err)
		return "", err
	}
	return app.App.Scheduler.DingTalkClient.Workflow.CreateProcessInstance(&dt.CreateProcessInstanceInput{
		ProcessCode:         app.App.Config.WithDingtalk.ProcessCode,
		OriginatorUserID:    tea.StringValue(user.DingtalkID),
		DeptId:              tea.StringValue(user.DingtalkDeptID),
		FormComponentValues: values,
	}, app.App.Scheduler.DingTalkClient.AccessToken.Token)
}

// 同步钉钉用户到数据库user表
func LoadUsers() error {
	err := app.App.Scheduler.DingTalkClient.SetAccessToken()
	if err != nil {
		log.Errorf("SetAccessToken failed! %s", err)
	}
	departIDChan := make(chan int64, 2000)
	go getDepart(departIDChan)
	startTime := time.Now()
	log.Infof("load dingtalk users start")
	for depart := range departIDChan {
		log.Infof("load depart %d", depart)
		departRes, err := app.App.Scheduler.DingTalkClient.Depart.GetDepartmentIDs(&dt.GetDepartmentsIDInput{
			DeptID: depart,
		}, app.App.Scheduler.DingTalkClient.AccessToken.Token)
		if err != nil {
			return err
		}
		for _, v := range departRes {
			departIDChan <- v
		}
		_users, err := app.App.Scheduler.DingTalkClient.User.GetUsers(&dt.GetUsersInput{
			DeptID: depart,
			Size:   100,
			Cursor: 0,
		}, app.App.Scheduler.DingTalkClient.AccessToken.Token)
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
	departRes, err := app.App.Scheduler.DingTalkClient.Depart.GetDepartmentIDs(input, app.App.Scheduler.DingTalkClient.AccessToken.Token)
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
	err := app.App.Scheduler.DingTalkClient.SetAccessToken() // 更新 token
	if err != nil {
		log.Errorf("SetAccessToken failed! %s", err)
		return
	}
	// 获取审批列表
	policies, err := app.App.DBIo.QueryAllPolicy()
	if err != nil {
		log.Errorf("QueryAllPolicy failed! %s", err)
		return
	}
	successes := syncApprovals(policies, app.App.Scheduler.DingTalkClient.Workflow, app.App.Scheduler.DingTalkClient.AccessToken.Token)
	log.Infof("load dingtalk approval success, %d/%d, cost %v", len(successes), len(policies), time.Since(timeStart))
}

// syncApprovals leaves policies unchanged when the upstream result is unavailable.
// A rate limit ends this poll; the next scheduled poll may try again without a retry burst.
func syncApprovals(policies []Policy, workflow dt.Workflow, token string) []string {
	var successes []string
	for _, policy := range policies {
		if policy.ApprovalID == "" || strings.Contains(policy.Approver, "BusinessId") {
			continue
		}
		resp, err := workflow.GetProcessInstance(policy.ApprovalID, token)
		if err != nil {
			log.Errorf("GetProcessInstance failed! %s", err)
			message := strings.ToLower(err.Error())
			if strings.Contains(message, "qpslimit") || strings.Contains(message, "throttl") || strings.Contains(message, "status code: 429") {
				break
			}
			continue
		}
		if resp == nil || !resp.Success {
			continue
		}
		result := resp.Result
		businessID := tea.StringValue(result.BusinessId)
		update := ApprovalResult{IsPass: tea.Bool(false)}
		switch tea.StringValue(result.Status) {
		case "COMPLETED":
			if businessID == "" || tea.StringValue(result.Result) == "" {
				continue
			}
			update.Applicant = tea.String("BusinessId: " + businessID)
			update.IsPass = tea.Bool(tea.StringValue(result.Result) == "agree")
		case "TERMINATED":
			update.Applicant = tea.String("BusinessId: TERMINATED")
		default:
			continue
		}
		if err := app.App.DBIo.UpdatePolicyStatus(policy.ID, update); err != nil {
			log.Errorf("UpdatePolicyStatus failed! %s", err)
			continue
		}
		successes = append(successes, businessID)
		log.Infof("update dingtalk approval %s(%s) to %v from DTCloud", businessID, tea.StringValue(result.Title), tea.BoolValue(update.IsPass))
	}
	return successes
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
