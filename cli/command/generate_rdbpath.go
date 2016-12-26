package command

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/codegangsta/cli"

	"github.com/ksarch-saas/cc/cli/context"
	"github.com/ksarch-saas/cc/controller/command"
	"github.com/ksarch-saas/cc/frontend/api"
	"github.com/ksarch-saas/cc/redis"
	"github.com/ksarch-saas/cc/topo"
	"github.com/ksarch-saas/cc/utils"
)

var GenerateRdbPathCommand = cli.Command{
	Name:   "genrdb",
	Usage:  "genrdb -r [bj/nj/gz] -s",
	Action: genRdbAction,
	Flags: []cli.Flag{
		cli.StringFlag{"r,region", "", "region to gernate rdb"},
		cli.BoolFlag{"s,save", "send bgsave command"},
	},
	Description: `
	generate rdb files for each replica set
	`,
}

func genRdbAction(c *cli.Context) {
	region := c.String("r")
	if region == "" {
		fmt.Println("region should be assigned")
		return
	}
	save := c.Bool("s")

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
	sort.Sort(topo.ByMasterId(rss.ReplicaSets))
	sort.Sort(topo.ByNodeState(rss.ReplicaSets))

	for _, rs := range rss.ReplicaSets {
		//slaves
		for _, n := range rs.Slaves {
			if n.Region == region {
				if save {
					//send bgsave command first, ignore result
					redis.RedisCli(n.Addr(), "BGSAVE")
				}
				unitId := strings.Split(n.Tag, ".")[1]
				rdbPath := fmt.Sprintf("ftp://%s/home/matrix/containers/%s.redis3db-%s.osp.%s/home/work/data/dump.rdb",
					n.Ip, unitId, context.GetAppName(), n.Zone)
				fmt.Println(rdbPath)
				break
			}
		}
	}
}
