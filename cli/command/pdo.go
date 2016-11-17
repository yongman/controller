package command

import (
	"fmt"
	"time"

	"github.com/codegangsta/cli"

	"github.com/ksarch-saas/cc/cli/context"
	"github.com/ksarch-saas/cc/controller/command"
	"github.com/ksarch-saas/cc/frontend/api"
	"github.com/ksarch-saas/cc/redis"
	"github.com/ksarch-saas/cc/utils"
)

var PdoCommand = cli.Command{
	Name:   "pdo",
	Usage:  "WANRNING: Send command to all nodes",
	Action: pdoAction,
	Description: `
	send command to all nodes
	`,
}

func pdoAction(c *cli.Context) {
	addr := context.GetLeaderAddr()
	url := "http://" + addr + api.FetchReplicaSetsPath

	resp, err := utils.HttpGet(url, nil, 5*time.Second)
	if err != nil {
		fmt.Println(err)
		return
	}

	var rss command.FetchReplicaSetsResult
	err = utils.InterfaceToStruct(resp.Body, &rss)
	if err != nil {
		fmt.Println(err)
		return
	}

	cmd := c.Args()[0]
	var args []interface{}
	for _, arg := range c.Args()[1:] {
		args = append(args, arg)
	}

	totalNodes := 0

	resChan := make(chan string, 2048)

	for _, rs := range rss.ReplicaSets {
		totalNodes += len(rs.AllNodes())
		for _, n := range rs.AllNodes() {
			go func(addr string) {
				res, _ := redis.RedisCli(addr, cmd, args...)
				ret := fmt.Sprintf("%s %s", addr, res)
				resChan <- ret
			}(n.Addr())
		}
	}
	for i := 0; i < totalNodes; i++ {
		ret := <-resChan
		fmt.Println(ret)
	}
	fmt.Println("Total nodes:", totalNodes)

}
