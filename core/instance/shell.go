package instance

import (
	"fmt"
	"sync"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/robfig/cron"
	"github.com/xops-infra/jms/app"
	. "github.com/xops-infra/jms/config"
	"github.com/xops-infra/noop/log"
)

// 查询数据库的批量脚本任务，符合条件后开始执行
func ServerShellRun() {
	// 查库
	tasks, err := app.App.DBService.ListShellTask()
	if err != nil {
		log.Errorf("list shell task error: %s", err)
	}
	wg := sync.WaitGroup{}
	for _, task := range tasks {
		log.Debugf("shell task: %s", tea.Prettify(task))
		if task.Status == StatusPending {
			// 状态更新
			err = app.App.DBService.UpdateShellTaskStatus(task.UUID, StatusRunning, "")
			if err != nil {
				log.Errorf("update shell task status error: %s", err)
				continue
			}
			wg.Add(1)
			go func(task ShellTask) {
				defer wg.Done()
				// 执行
				startTime := time.Now()

				log.Infof("shell task start: %s", task.UUID)
				status, err := runShellTask(task)
				if err != nil {
					log.Errorf("run shell task error: %s", err)
					app.App.DBService.UpdateShellTaskStatus(task.UUID, status, err.Error())

				}
				app.App.DBService.UpdateShellTaskStatus(task.UUID, StatusSuccess, "")
				log.Infof("shell task %s finished, cost: %s", task.UUID, time.Since(startTime))
			}(task)
		}
	}
	wg.Wait()
	log.Infof("shell task finished")
}

func runShellTask(task ShellTask) (Status, error) {
	servers := GetServers()
	for _, server := range servers {
		fmt.Println(tea.Prettify(server))
	}

	return StatusSuccess, nil
}

// corn任务的处理，实现对 corn 的支持，主要就是判断时间对了就修改一下任务状态
func ServerCronRun() {
	tasks, err := app.App.DBService.ListShellTask()
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
		err = app.App.DBService.UpdateShellTaskStatus(task.UUID, StatusPending, "system reset pengding cause cron time match")
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
