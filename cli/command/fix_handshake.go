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

var FixHandshakeCommand = cli.Command{
	Name:   "fixhandshake",
	Usage:  "fix handshake",
	Action: fixHandshakeAction,
	Description: `
	fix handshake in cluster
	`,
}

func fixHandshakeAction(c *cli.Context) {
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

	seedNodes := []string{}
	allFailedNodes := []string{}
	inner := func(addr string) {
		failedNodes, err := getFailedNodes(addr)
		if err != nil {
			fmt.Println(err)
			return
		}
		if len(failedNodes) > 0 {
			seedNodes = append(seedNodes, addr)
		}
		for _, fn := range failedNodes {
			if nodeExists(fn, allFailedNodes) == false {
				allFailedNodes = append(allFailedNodes, fn)
			}
		}
	}
	for _, rs := range rss.ReplicaSets {
		n := rs.Master
		inner(n.Addr())

		//slaves
		for _, ns := range rs.Slaves {
			inner(ns.Addr())
		}
	}

	//send forget to need fix nodes
	resChan := make(chan string, len(seedNodes))
	for _, failed := range allFailedNodes {
		for _, seed := range seedNodes {
			go func(seed, failed string) {
				resp, err := redis.ClusterForget(seed, failed)
				res := fmt.Sprintf("Node %s forget %s result %s %v", seed, failed, resp, err)
				resChan <- res
			}(seed, failed)
		}
		for i := 0; i < len(seedNodes); i++ {
			res := <-resChan
			fmt.Println(res)
		}
	}
}

func nodeExists(addr string, nodes []string) bool {
	for _, node := range nodes {
		if node == addr {
			return true
		}
	}
	return false
}

func getFailedNodes(addr string) ([]string, error) {
	resp, err := redis.ClusterNodes(addr)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(resp, "\n")
	failedNodes := []string{}
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		node, _ := checkNodeStatus(line)
		if node.Fail {
			failedNodes = append(failedNodes, node.Id)
		}
	}

	return failedNodes, nil
}

func checkNodeStatus(line string) (*topo.Node, bool) {
	xs := strings.Split(line, " ")
	_, _, id, addr, flags, parent := xs[0], xs[1], xs[2], xs[3], xs[4], xs[5]
	node := topo.NewNodeFromString(addr)
	// basic info
	node.SetId(id)
	node.SetParentId(parent)
	handshake := false
	if strings.Contains(flags, "handshake") {
		handshake = true
	}
	if strings.Contains(flags, "fail?") {
		node.SetFail(true)
	}
	return node, handshake
}
