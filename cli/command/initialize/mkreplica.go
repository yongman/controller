package initialize

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/codegangsta/cli"
	"github.com/fatih/color"
	"github.com/ksarch-saas/cc/cli/context"
	"github.com/ksarch-saas/cc/controller/command"
	"github.com/ksarch-saas/cc/frontend/api"
	"github.com/ksarch-saas/cc/topo"
	"github.com/ksarch-saas/cc/utils"
)

var (
	flags_mkreplica = []cli.Flag{
		cli.StringFlag{"l,logic", "", "logic machine rooms list"},
		cli.StringFlag{"m,master", "", "master machine rooms list"},
		cli.IntFlag{"r,replicas", 0, "replicaset of each master node"},
	}
	// meet cluster and make replicaset
	MakeReplicaCommand = cli.Command{
		Name:   "mkreplicasets",
		Usage:  "make replicasets and add to cluster already exists",
		Action: mkreplicaAction,
		Flags:  flags_mkreplica,
	}
)

func mkreplicaAction(c *cli.Context) {

	red := color.New(color.FgRed).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	l := c.String("l")
	if l == "" {
		fmt.Println(red("-l logic machine room must be assigned"))
		os.Exit(-1)
	}
	m := c.String("m")
	if m == "" {
		fmt.Println(red("-m master machine rooms must be assigned"))
		os.Exit(-1)
	}
	replicas := c.Int("r")
	masterRooms := strings.Split(m, ",")
	rooms := strings.Split(l, ",")

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

	freeNodes := []*Node{}

	for _, rs := range rss.ReplicaSets {
		if rs.Master != nil && len(rs.Master.Ranges) == 0 && len(rs.Slaves) == 0 {
			// this is a free node
			node := &Node{
				Ip:      rs.Master.Ip,
				Port:    fmt.Sprintf("%d", rs.Master.Port),
				LogicMR: rs.Master.Zone,
			}
			freeNodes = append(freeNodes, node)
		}
	}
	// get all free nodes done
	if replicas != 0 && len(freeNodes)%(replicas+1) != 0 {
		fmt.Printf("%s. Not enough nodes\n", red("ERROR"))
		return
	}

	// check free nodes state
	for _, fn := range freeNodes {
		fn.Alive = isAlive(fn)
		fmt.Printf("connecting to %s\t%s\t", fn.Ip, fn.Port)
		if fn.Alive {
			fmt.Printf("%s\n", green("OK"))
		} else {
			fmt.Printf("%s\n", red("FAILED"))
		}
	}

	// check and set state
	fmt.Println("Check and set state...")
	for _, fn := range freeNodes {
		err := checkAndSetState(fn)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	// validate
	if validateProcess(freeNodes) == false {
		fmt.Println("Not all nodes have the right status")
		return
	}

	// build replicas
	fmt.Println("Build replicas...")
	masterNodes, err := buildCluster(freeNodes, replicas, masterRooms, rooms)
	if err != nil {
		fmt.Println("build cluster failed, ", err)
		return
	}

	// summary
	for _, mn := range masterNodes {
		fmt.Printf("%s %s\t%s\t%s\t%s\n", yellow("M:"), mn.Id, mn.Ip, mn.Port, yellow(mn.SlotsRange))
		slaves := getSlaves(freeNodes, mn)
		for _, slave := range slaves {
			fmt.Printf("%s %s\t%s\t%s\t%s\n", cyan("S:"), slave.Id, slave.Ip, slave.Port, slave.MasterId)
		}
	}
	var cmd string
	fmt.Printf("Type %s to continue: ", green("yes"))
	fmt.Printf("%s\n", red("(--force will reset the cluster)"))

	fmt.Scanf("%s\n", &cmd)
	if cmd != "yes" {
		os.Exit(0)
	}
	meetEach(freeNodes)

	for _, mn := range masterNodes {
		fmt.Printf("Node:%s\n", mn.Id)
		resp, err := rwMasterState(mn)
		if err != nil {
			fmt.Printf("%s\n", red("FAILED to chmod, please check"))
		}
		slaves := getSlaves(freeNodes, mn)
		fmt.Printf("%-40s", "setting replicas...")
		err = rwReplicasState(slaves)
		if err != nil {
			fmt.Printf("%s\n", red("FAILED to chmod, please check"))
		}
		resp, err = setReplicas(slaves)
		if err != nil {
			fmt.Printf("%s\n", red(err.Error()))
			break
		} else {
			fmt.Printf("%s\n", green(resp))
		}
	}
}
