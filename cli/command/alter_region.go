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
	"github.com/ksarch-saas/cc/topo"
	"github.com/ksarch-saas/cc/utils"
)

var AlterRegionCommand = cli.Command{
	Name:   "alterRegion",
	Usage:  "alterRegion nj/bj",
	Action: alterRegionAction,
	Flags: []cli.Flag{
		cli.StringFlag{"r,region", "", "new master region"},
	},
	Description: `
	change master region
	`,
}

func getRegion(node *topo.Node) string {
	tag := node.Tag
	if !strings.Contains(tag, ":") {
		fmt.Println("Err: Tag invalid")
		return ""
	}
	return strings.Split(tag, ":")[0]
}

func alterRegionAction(c *cli.Context) {
	region := c.String("r")
	if region == "" {
		fmt.Println(ErrInvalidParameter)
		return
	}
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

	var old_master *topo.Node
	var new_master *topo.Node
	var new_slaves []*topo.Node

	for _, rs := range rss.ReplicaSets {
		old_master = rs.Master
		if old_master.IsArbiter() {
			continue
		}
		old_master_region := getRegion(old_master)
		new_master_region := ""
		if old_master_region == "" {
			return
		}

		//already altered to new region, just ignore
		if old_master_region == region {
			continue
		}

		new_slaves = append(new_slaves, old_master)

		// choose a new master
		skip := false
		for _, repli := range rs.Slaves {
			new_master_region = getRegion(repli)
			if new_master_region == "" {
				return
			}
			if new_master_region == region && !skip {
				new_master = repli
				skip = true
			} else {
				new_slaves = append(new_slaves, repli)
			}
		}
		if new_master == nil {
			fmt.Printf("Select new master failed")
			return
		}

		//disable read flag of the all new slaves,including old master
		for _, s := range new_slaves {
			resp, err = configRead(s, false)
			if err != nil {
				fmt.Println(err)
				return
			}
			if resp.Errno != 0 {
				fmt.Println(resp.Errmsg)
				return
			}
		}

		//send failover command to new_master
		extraHeader := &utils.ExtraHeader{
			User:  context.Config.User,
			Role:  context.Config.Role,
			Token: context.Config.Token,
		}
		url_setmaster := "http://" + addr + api.NodeSetAsMasterPath
		req_setmaster := api.FailoverTakeoverParams{
			NodeId: new_master.Id,
		}

		fmt.Printf("New master: R[%s] IP[%s] Port[%d]\n", getRegion(new_master), new_master.Ip, new_master.Port)

		fmt.Printf("Old master: R[%s] IP[%s] Port[%d]\n", old_master_region, old_master.Ip, old_master.Port)

		resp, err = utils.HttpPostExtra(url_setmaster, req_setmaster, 5*time.Second, extraHeader)
		if err != nil {
			fmt.Println(err)
			return
		}
		if resp.Errno != 0 {
			fmt.Println(resp.Errmsg)
			return
		}

		//check the status of all slaves
		cnt := 1
		for {
			fmt.Printf("Check slaves status %d times\n", cnt)
			cnt++
			inner := func(nodes []*topo.Node) bool {
				rok := true
				for _, n := range nodes {
					ok, err := checkSlaveRepliStatusOk(n)
					if ok {
						//replica status ok,enable read flag,ignore result
						configRead(n, true)
						continue
					}
					if !ok || err != nil {
						rok = false
					}
				}
				return rok
			}

			ok := inner(new_slaves)
			if !ok {
				//not ok, wait for next turn check
				time.Sleep(10 * time.Second)
			} else {
				break
			}
		}
	}

}
