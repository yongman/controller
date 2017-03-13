package command

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/codegangsta/cli"
	"github.com/ksarch-saas/cc/topo"

	"github.com/ksarch-saas/cc/redis"
)

var TopoCommand = cli.Command{
	Name:   "topo",
	Usage:  "get cluster topo from seed node",
	Action: topoAction,
	Flags: []cli.Flag{
		cli.StringFlag{"s,seed", "", "seed fetch cluster nodes from"},
	},
	Description: `
	return cluster topo
	`,
}

func buildNode(line string) (*topo.Node, error) {
	xs := strings.Split(line, " ")
	mod, tag, id, addr, flags, parent := xs[0], xs[1], xs[2], xs[3], xs[4], xs[5]
	node := topo.NewNodeFromString(addr)
	ranges := []string{}
	for _, word := range xs[10:] {
		if strings.HasPrefix(word, "[") {
			word = word[1 : len(word)-1]
			xs := strings.Split(word, "->-")
			if len(xs) == 2 {
				slot, _ := strconv.Atoi(xs[0])
				node.AddMigrating(xs[1], slot)
			}
			xs = strings.Split(word, "-<-")
			if len(xs) == 2 {
				slot, _ := strconv.Atoi(xs[0])
				node.AddImporting(xs[1], slot)
			}
			continue
		}
		ranges = append(ranges, word)
	}

	for _, r := range ranges {
		xs = strings.Split(r, "-")
		if len(xs) == 2 {
			left, _ := strconv.Atoi(xs[0])
			right, _ := strconv.Atoi(xs[1])
			node.AddRange(topo.Range{left, right})
		} else {
			left, _ := strconv.Atoi(r)
			node.AddRange(topo.Range{left, left})
		}
	}

	// basic info
	node.SetId(id)
	node.SetParentId(parent)
	node.SetTag(tag)
	node.SetReadable(mod[0] == 'r')
	node.SetWritable(mod[1] == 'w')
	if strings.Contains(flags, "master") {
		node.SetRole("master")
	} else if strings.Contains(flags, "slave") {
		node.SetRole("slave")
	}
	if strings.Contains(flags, "noaddr") {
		return nil, fmt.Errorf("cluster nodes contains noaddr")
	}
	if strings.Contains(flags, "fail?") {
		node.SetPFail(true)
	}
	xs = strings.Split(tag, ":")
	if len(xs) == 3 {
		node.SetRegion(xs[0])
		node.SetZone(xs[1])
		node.SetRoom(xs[2])
	}

	return node, nil
}

func initTopo(addr string) (*topo.Cluster, error) {
	resp, err := redis.ClusterNodes(addr)
	if err != nil {
		return nil, err
	}

	cluster := topo.NewCluster("topo")
	lines := strings.Split(resp, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		node, err := buildNode(line)
		if err != nil {
			return nil, err
		}
		cluster.AddNode(node)
	}
	return cluster, nil
}

func topoAction(c *cli.Context) {
	seed := c.String("seed")
	fmt.Println("seed addr is ", seed)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', tabwriter.Debug)

	cluster, err := initTopo(seed)
	if err != nil {
		fmt.Println(err)
	}
	cluster.BuildReplicaSets()
	for _, rs := range cluster.ReplicaSets() {
		PrintNode(w, rs.Master)
		for _, s := range rs.Slaves {
			PrintNode(w, s)
		}
		fmt.Fprintln(w, "========================")
	}
	w.Flush()
}
func PrintNode(w io.Writer, node *topo.Node) {
	fmt.Fprintln(w, node.Id+" \t "+node.Role+" \t "+node.Addr()+" \t "+node.Tag)
}
