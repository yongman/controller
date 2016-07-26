package command

import (
	"fmt"

	cc "github.com/ksarch-saas/cc/controller"
	"github.com/ksarch-saas/cc/meta"
)

type MigrateRecoverCommand struct {
	ShowOnly bool
}

// Rebalance任务同时只能有一个
func (self *MigrateRecoverCommand) Execute(c *cc.Controller) (cc.Result, error) {
	mm := c.MigrateManager
	cs := c.ClusterState
	cluster := cs.GetClusterSnapshot()

	migrateMetas, err := meta.AllMigrateTasks()
	if err != nil {
		return nil, err
	}

	if !self.ShowOnly {
		mm.RebuildTasks(migrateMetas, cluster)
	}
	var tasks []string
	for _, m := range migrateMetas {
		task := fmt.Sprintf("From:%s To:%s Slot:%s", m.SourceId, m.TargetId, m.Ranges)
		tasks = append(tasks, task)
	}

	return tasks, nil
}
