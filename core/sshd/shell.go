package sshd

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/robfig/cron"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/jms/core/dingtalk"
	. "github.com/xops-infra/jms/model"
	"github.com/xops-infra/noop/log"
)

// 查询数据库的批量脚本任务，符合条件后开始执行
func ServerShellRun() {
	// 查库
	tasks, err := app.App.JmsDBService.ListShellTask()
	if err != nil {
		log.Errorf("list shell task error: %s", err)
	}
	// servers := app.GetServers()
	wg := sync.WaitGroup{}
	for _, task := range tasks {
		log.Debugf("shell task: %s", tea.Prettify(task))
		if task.Status == StatusPending {
			// 状态更新
			err = app.App.JmsDBService.UpdateShellTaskStatus(task.UUID, StatusRunning, "")
			if err != nil {
				log.Errorf("update shell task status error: %s", err)
				continue
			}
			wg.Add(1)
			go func(task ShellTask) {
				startTime := time.Now()
				state := StatusSuccess
				result := ""
				defer func() {
					log.Debugf("shell task done: %s, state: %s, result: %s", task.UUID, state, result)
					err := app.App.JmsDBService.UpdateShellTaskStatus(task.UUID, state, result)
					if err != nil {
						log.Errorf("update shell task status error: %s", err)
					}
					// 发送任务执行完成通知
					err = dingtalk.SendRobotText(os.Getenv("JMS_DINGTALK_WEB_HOOK_TOKEN"), fmt.Sprintf("shell task %s(%s) status:%s  %s", task.Name, task.UUID, state, result), "")
					if err != nil {
						log.Errorf("send dingtalk error: %s", err)
					}
					wg.Done()
				}()

				// 执行
				log.Infof("shell task start: %s", task.UUID)
				status, err := RunShellTask(task, app.GetServers())
				if err != nil {
					log.Errorf("run shell task error: %s", err)
					state = status
					result = err.Error()
					return
				}
				state = status
				result = fmt.Sprintf("finished, cost: %s", time.Since(startTime))
				log.Infof("shell task %s finished, cost: %s", task.UUID, time.Since(startTime))
			}(task)
		}
	}
	wg.Wait()
	log.Infof("shell task finished")
}

func RunShellTask(task ShellTask, servers Servers) (Status, error) {

	wg := sync.WaitGroup{}

	faildServers := []string{}
	totalServer := 0
	for _, server := range servers {
		if MatchServerByFilter(task.Servers, server, false) {
			totalServer++
			wg.Add(1)
			log.Debugf("shell task: %s, cmd: %s, run on server: %s", task.UUID, task.Shell, server.Host)
			go func(server Server) {
				defer func() {
					wg.Done()
				}()
				// 执行
				if err := runShell(server, task); err != nil {
					log.Errorf("server %s run shell error: %s", server.Host, err)
					faildServers = append(faildServers, server.Host)
					return
				}
				log.Infof("server %s run shell success: %s", server.Host, task.Shell)
			}(server)
		} else {
			log.Debugf("server %s not match filter", server.Host)
		}
	}
	wg.Wait()

	if len(faildServers) > 0 {
		if len(faildServers) == totalServer {
			return StatusFailed, fmt.Errorf("all servers failed")
		}
		return StatusNotAllSuccess, fmt.Errorf("some servers failed: %s", faildServers)
	}

	if totalServer == 0 {
		return StatusFailed, fmt.Errorf("not found servers")
	}

	return StatusSuccess, nil
}

func runShell(server Server, task ShellTask) error {
	req := &CreateShellTaskRecordRequest{
		TaskID:     tea.String(task.UUID),
		TaskName:   tea.String(task.Name),
		ExecTimes:  tea.Int(task.ExecTimes + 1),
		ServerName: tea.String(server.Name),
		ServerIP:   tea.String(server.Host),
		Shell:      tea.String(task.Shell),
	}

	execStartTime := time.Now()

	defer func() {
		req.CostTime = tea.String(time.Since(execStartTime).String())
		log.Debugf("shell task record: %s", tea.Prettify(req))
		err := app.App.JmsDBService.CreateShellTaskRecord(req)
		if err != nil {
			log.Errorf("create shell task record error: %s", err)
		}
	}()

	if len(server.SSHUsers) == 0 {
		return fmt.Errorf("server %s has no ssh user", server.Host)
	}

	sshUser := server.SSHUsers[0]
	proxyClient, client, err := NewSSHClient(server, sshUser)
	if err != nil {
		req.IsSuccess = tea.Bool(false)
		req.Output = tea.String(err.Error())
		return err
	}
	if proxyClient != nil {
		defer proxyClient.Close()
	}
	defer client.Close()
	sess, _ := client.NewSession()
	defer sess.Close()

	// 执行命令
	info, err := sess.Output(task.Shell)
	if err != nil {
		req.IsSuccess = tea.Bool(false)
		req.Output = tea.String(string(info))
		return err
	}
	req.IsSuccess = tea.Bool(true)
	req.Output = tea.String(string(info))
	return nil
}

// corn任务的处理，实现对 corn 的支持，主要就是判断时间对了就修改一下任务状态
func ServerCronRun() {
	tasks, err := app.App.JmsDBService.ListShellTask()
	if err != nil {
		log.Errorf("list shell task error: %s", err)
	}
	for _, task := range tasks {
		if task.Corn == "" || task.Status == StatusRunning {
			continue
		}
		// 校验时间
		if !checkCronTime(task.Corn) {
			continue
		}
		// 更新任务状态
		err = app.App.JmsDBService.UpdateShellTaskStatus(task.UUID, StatusPending, "system reset pengding cause cron time match")
		if err != nil {
			log.Errorf("update shell task status error: %s", err)
		}
	}
}

func checkCronTime(cronExpr string) bool {
	parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		log.Errorf("parse cron expression error: %s", err)
		return false
	}

	nextRun := schedule.Next(time.Now().Add(-1 * time.Minute))
	return nextRun.Before(time.Now()) && nextRun.After(time.Now().Add(-1*time.Minute))
}
