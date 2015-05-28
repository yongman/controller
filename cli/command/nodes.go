package command

import (
	"fmt"
	"sort"
	"time"

	"github.com/jxwr/cc/cli/context"
	"github.com/jxwr/cc/controller/command"
	"github.com/jxwr/cc/frontend/api"
	"github.com/jxwr/cc/topo"
	"github.com/jxwr/cc/utils"
)

type RNode struct {
	Id         string
	ParentId   string
	Role       string
	Addr       string
	Fail       string
	Mode       string
	Tag        string
	Keys       int64
	Repl       string
	Link       string
	QPS        int
	NetIn      string
	NetOut     string
	UsedMemory string
}

func toReadable(node *topo.Node) *RNode {
	if node == nil {
		return nil
	}
	n := &RNode{
		Id:       node.Id,
		ParentId: node.ParentId,
		Tag:      node.Tag,
		Role:     "S",
		Fail:     "OK",
		Mode:     "--",
		Addr:     fmt.Sprintf("%s:%d", node.Ip, node.Port),
		Keys:     node.SummaryInfo.Keys,
		Link:     node.SummaryInfo.MasterLinkStatus,
		QPS:      node.SummaryInfo.InstantaneousOpsPerSec,
	}

	if node.Role == "master" {
		n.Role = "M"
	}
	if node.Fail {
		n.Fail = "Fail"
	}
	if node.Readable && node.Writable {
		n.Mode = "rw"
	}
	if node.Readable && !node.Writable {
		n.Mode = "r-"
	}
	if !node.Readable && node.Writable {
		n.Mode = "-w"
	}
	if node.IsMaster() {
		n.Link = "up"
	}
	n.UsedMemory = fmt.Sprintf("%0.2fG", float64(node.SummaryInfo.UsedMemory)/1024.0/1024.0/1024.0)
	n.NetIn = fmt.Sprintf("%.2fKbps", node.SummaryInfo.InstantaneousInputKbps)
	n.NetOut = fmt.Sprintf("%.2fKbps", node.SummaryInfo.InstantaneousOutputKbps)
	n.Repl = fmt.Sprintf("%d", node.ReplOffset)
	return n
}

func toInterfaceSlice(nodes []*topo.Node) []interface{} {
	var interfaceSlice []interface{} = make([]interface{}, len(nodes))
	for i, node := range nodes {
		interfaceSlice[i] = toReadable(node)
	}
	return interfaceSlice
}

func showNodes() {
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

	var allNodes []*topo.Node
	for i, rs := range rss.ReplicaSets {
		allNodes = append(allNodes, rs.Master)
		for _, node := range rs.Slaves {
			allNodes = append(allNodes, node)
		}
		if i < len(rss.ReplicaSets)-1 {
			allNodes = append(allNodes, nil)
		}
	}

	utils.PrintJsonArray("table",
		[]string{"Mode", "Fail", "Role", "Id", "Tag", "Addr", "QPS",
			"UsedMemory", "Link", "Repl", "Keys", "NetIn", "NetOut"},
		toInterfaceSlice(allNodes))
}

func showSlots() {
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

	for _, rs := range rss.ReplicaSets {
		fmt.Println("  ", rs.Master.Id, rs.Master.Ranges)
	}
}
