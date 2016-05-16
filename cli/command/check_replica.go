package command

import (
	"fmt"
	"sort"
	"time"

	"github.com/codegangsta/cli"

	"github.com/ksarch-saas/cc/cli/context"
	"github.com/ksarch-saas/cc/controller/command"
	"github.com/ksarch-saas/cc/frontend/api"
	"github.com/ksarch-saas/cc/topo"
	"github.com/ksarch-saas/cc/utils"
)

var CheckReplicaCommand = cli.Command{
	Name:   "checkreplica",
	Usage:  "check nodes of one replicasets",
	Action: checkReplicaAction,
	Description: `
	check node of one replicasets in cluster
	`,
}

func checkReplicaAction(c *cli.Context) {
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
		hostMap := map[string]bool{}

		n := rs.Master
		hostMap[n.Ip] = true

		//slaves
		for _, ns := range rs.Slaves {
			if _, ok := hostMap[ns.Ip]; ok {
				fmt.Printf("[%s] %s:%d %s\n", ns.Region, n.Ip, n.Port, "Replica has nodes in same host")
			} else {
				hostMap[ns.Ip] = true
			}
		}
	}
	fmt.Println("Check done")
}
