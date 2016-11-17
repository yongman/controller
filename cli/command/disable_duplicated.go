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

var DisableduplicatedCommand = cli.Command{
	Name:   "disableduplicated",
	Usage:  "disabledeplicated [-r bj/nj/gz] [-z jx/tc/nj/nj03/gz] -l 1",
	Action: disableDuplicatedAction,
	Flags: []cli.Flag{
		cli.StringFlag{"r,region", "", "region to disable"},
		cli.StringFlag{"z,zone", "", "zone to disable"},
		cli.IntFlag{"l,limit", 2, "number of nodes to reserve in a partition in region/zone"},
	},
	Description: `
	disable read of duplicated slave servers
	`,
}

func disableDuplicatedAction(c *cli.Context) {
	region := c.String("r")
	zone := c.String("z")
	if region == "" && zone == "" {
		fmt.Println("region or zone should be assigned")
		return
	}
	if region != "" && zone != "" {
		fmt.Println("region or zone should be choose one")
		return
	}
	limit := c.Int("l")
	if limit < 1 {
		fmt.Println("limit should be >=1 ")
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
	extraHeader := &utils.ExtraHeader{
		User:  context.Config.User,
		Role:  context.Config.Role,
		Token: context.Config.Token,
	}
	url = "http://" + addr + api.NodePermPath
	for _, rs := range rss.ReplicaSets {
		rlimit := limit
		if region != "" {
			n := rs.Master
			if n.Region == region {
				rlimit--
			}

			//slaves
			for _, ns := range rs.Slaves {
				if rlimit <= 0 && ns.Region == region {
					//chmod -r
					fmt.Printf("Disable node addr: %s region: %s\n", ns.Addr(), ns.Region)

					req := api.ToggleModeParams{
						NodeId: ns.Id,
						Action: "disable",
						Perm:   "read",
					}
					resp, err := utils.HttpPostExtra(url, req, 5*time.Second, extraHeader)
					if err != nil {
						fmt.Println(err)
						return
					}
					ShowResponse(resp)
				}
				if ns.Region == region {
					rlimit--
				}
			}
		} else {
			n := rs.Master
			if n.Zone == zone {
				rlimit--
			}

			//slaves
			for _, ns := range rs.Slaves {
				if rlimit <= 0 && ns.Zone == zone {
					//chmod -r
					fmt.Printf("Disable node addr: %s zone: %s\n", ns.Addr(), ns.Zone)

					req := api.ToggleModeParams{
						NodeId: ns.Id,
						Action: "disable",
						Perm:   "read",
					}
					resp, err := utils.HttpPostExtra(url, req, 5*time.Second, extraHeader)
					if err != nil {
						fmt.Println(err)
						return
					}
					ShowResponse(resp)
				}
				if ns.Zone == zone {
					rlimit--
				}
			}
		}
	}
}
