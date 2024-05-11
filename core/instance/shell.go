package instance

import (
	"github.com/alibabacloud-go/tea/tea"
	"github.com/xops-infra/jms/app"
	"github.com/xops-infra/noop/log"
)

// 查询数据库的批量脚本任务，符合条件后开始执行
func ServerShellRun() {
	for {
		// 查库
		tasks, err := app.App.DBService.ListShellTask()
		if err != nil {
			log.Errorf("list shell task error: %s", err)
		}
		for _, task := range tasks {
			log.Debugf("shell task: %s", tea.Prettify(task))
		}
	}
}
