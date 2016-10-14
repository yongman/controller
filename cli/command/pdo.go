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
	fmt.Println(cmd)
	fmt.Println(args)

	//resChan := make(chan interface{}, 1000)
	totalNodes := 0

	for _, rs := range rss.ReplicaSets {
		totalNodes += len(rs.AllNodes())
		for _, n := range rs.AllNodes() {
			go func() {
				res, _ := redis.RedisCli(n.Addr(), cmd, args...)
				//resChan := <-res
				fmt.Printf("%s\r\n", res)
			}()
		}
	}

	for i := 0; i < totalNodes; i++ {
		//res := <-resChan
		//fmt.Println(res)
	}
}
