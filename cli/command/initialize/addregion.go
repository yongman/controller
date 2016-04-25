package initialize

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

var (
	flags_addregion = []cli.Flag{
		cli.StringFlag{"r,region", "", "regions u want to add"},
	}
	// meet cluster and make replicaset
	AddRegionCommand = cli.Command{
		Name:   "addregion",
		Usage:  "addregion to cluster already exists",
		Action: addRegionAction,
		Flags:  flags_addregion,
	}
)

func addRegionAction(c *cli.Context) {
	region := c.String("r")
	if region == "" {
		fmt.Println("-r region must be assigned")
		return
	}

	//fetch and check cluster nodes
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

	freeNodes := []*topo.Node{}
	masterNodes := []*topo.Node{}

	for _, rs := range rss.ReplicaSets {
		if rs.Master != nil && len(rs.Master.Ranges) == 0 && len(rs.Slaves) == 0 {
			// this is a free node
			if rs.Master.Region == region {
				freeNodes = append(freeNodes, rs.Master)
			}
		} else {
			masterNodes = append(masterNodes, rs.Master)
		}
	}

	if len(freeNodes)%len(masterNodes) != 0 {
		fmt.Println("Number of free nodes in region not match")
		return
	}
	repli_num := len(freeNodes) / len(masterNodes)
	// meet free nodes
	allNodes := append(masterNodes, freeNodes...)
	allNodes_alter := []*Node{}
	for _, n := range allNodes {
		node := &Node{
			Ip:   n.Ip,
			Port: fmt.Sprintf("%d", n.Port),
		}
		allNodes_alter = append(allNodes_alter, node)
	}
	meetEach(allNodes_alter)

	// set replica
	for idx, r := range masterNodes {
		slaves := []*Node{}
		for i := 0; i < repli_num; i++ {
			s := freeNodes[idx*repli_num+i]
			fmt.Printf("%s %s\n", "setting replicas", r.Id)
			node := &Node{
				Id:       s.Id,
				Ip:       s.Ip,
				Port:     fmt.Sprintf("%d", s.Port),
				MasterId: r.Id,
			}
			slaves = append(slaves, node)
		}
		err := rwReplicasState(slaves)
		if err != nil {
			fmt.Println(err)
		}
		resp, err := setReplicas(slaves)
		if err != nil {
			fmt.Println(err)
			break
		} else {
			fmt.Println(resp)
		}
	}

	if checkClusterInfo(allNodes_alter) {
		fmt.Println("All nodes agree the configure")
	} else {
		fmt.Println("Node configure inconsistent or slots incomplete")
	}
}
