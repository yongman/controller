package command

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/ksarch-saas/cc/redis"

	"github.com/ksarch-saas/cc/topo"
)

var TakeoverClusterCommand = cli.Command{
	Name:   "takeoverCluster",
	Usage:  "takeoverCluster -r=nj/bj/gz/hz/sh -s=seed",
	Action: takeoverClusterAction,
	Flags: []cli.Flag{
		cli.StringFlag{"s,seed", "", "seed to fetch cluster topo "},
		cli.StringFlag{"r,region", "", "new master region"},
		cli.BoolFlag{"without-check", "without check slaves status"},
	},
	Description: `
	change master region of the cluster
	Please make sure master have no data write, otherwise takeover will lost data
	`,
}

func takeoverClusterAction(c *cli.Context) {
	fmt.Printf("[Danger!!!] Type %s to continue: ", "yes")

	var cmd string
	fmt.Scanf("%s\n", &cmd)
	if cmd != "yes" {
		os.Exit(0)
	}
	withoutCheck := c.Bool("without-check")
	seed := c.String("seed")
	if seed == "" {
		fmt.Println(ErrInvalidParameter)
		return
	}
	fmt.Println("seed addr is ", seed)
	cluster, err := initTopo(seed)
	if err != nil {
		fmt.Println(err)
	}
	cluster.BuildReplicaSets()

	region := c.String("r")
	if region == "" {
		fmt.Println(ErrInvalidParameter)
		return
	}
	var old_master *topo.Node
	var new_master *topo.Node

	for _, rs := range cluster.ReplicaSets() {
		var new_slaves []*topo.Node
		old_master = rs.Master
		old_master_region := old_master.Region
		new_master_region := ""
		if old_master_region == "" {
			return
		}

		//already altered to new region, just ignore
		if old_master_region == region {
			continue
		}

		// choose a new master
		skip := false
		for _, repli := range rs.Slaves {
			new_master_region = repli.Region
			if new_master_region == "" {
				return
			}
			if new_master_region == region && !skip {
				new_master = repli
				skip = true
			} else {
				if repli.Region != old_master_region {
					new_slaves = append(new_slaves, repli)
				}
			}
		}
		if new_master == nil {
			fmt.Printf("Select new master failed")
			return
		}

		fmt.Printf("New master: R[%s] IP[%s] Port[%d]\n", new_master.Region, new_master.Ip, new_master.Port)
		fmt.Printf("Old master: R[%s] IP[%s] Port[%d]\n", old_master_region, old_master.Ip, old_master.Port)
		resp, err := TryFailover(new_master.Addr())
		fmt.Println("Trying failover...")
		if err != nil {
			fmt.Println(err)
			return
		}
		fmt.Println(resp)

		//check the status of all old slaves
		//old master may be fail
		cnt := 1
		for !withoutCheck {
			fmt.Printf("Check slaves status %d times\n", cnt)
			cnt++
			inner := func(nodes []*topo.Node) bool {
				rok := true
				for _, n := range nodes {
					ok, err := checkSlaveRepliStatusOk(n)
					if ok {
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
				time.Sleep(1 * time.Second)
			} else {
				break
			}
		}
	}

}

func TryFailover(addr string) (string, error) {
	resp, err := redis.ClusterFailover(addr, nil)
	if err != nil {
		if strings.HasPrefix(err.Error(), "ERR CLUSTERDOWN") {
			resp, err = redis.ClusterTakeover(addr, nil)
		}
	}
	return resp, err
}
