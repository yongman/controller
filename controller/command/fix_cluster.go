package command

import (
	"time"

	"errors"

	cc "github.com/ksarch-saas/cc/controller"
	"github.com/ksarch-saas/cc/log"
	"github.com/ksarch-saas/cc/state"
	"github.com/ksarch-saas/cc/topo"
)

type FixClusterCommand struct {
}

type FixClusterResult struct {
	Result bool
}

func (self *FixClusterCommand) Type() cc.CommandType {
	return cc.CLUSTER_COMMAND
}

func (self *FixClusterCommand) Execute(c *cc.Controller) (cc.Result, error) {
	cs := c.ClusterState
	snapshot := cs.GetClusterSnapshot()
	if snapshot == nil {
		return nil, nil
	}
	snapshot.BuildReplicaSets()

	nodeStates := map[string]string{}
	nss := cs.AllNodeStates()
	for id, n := range nss {
		nodeStates[id] = n.CurrentState()
	}
	rss := snapshot.ReplicaSets()

	totalNum := 0 //总节点数
	totalRepli := 0
	failedNodes := []*topo.Node{}
	freeNodes := []*topo.Node{}
	defectMaster := []*topo.Node{}

	for _, rs := range rss {
		//check failed nodes and free nodes
		if rs.Master != nil && rs.Master.IsArbiter() {
			continue
		}
		totalNum = totalNum + len(rs.AllNodes())
		totalRepli = totalRepli + 1
		if len(rs.AllNodes()) == 1 && nodeStates[rs.Master.Id] == state.StateRunning {
			//free节点
			freeNodes = append(freeNodes, rs.Master)
		} else {
			for _, node := range rs.AllNodes() {
				if nodeStates[node.Id] != state.StateRunning {
					failedNodes = append(failedNodes, node)
				}
			}
		}
	}
	if len(freeNodes) == 0 && len(failedNodes) == 0 {
		return nil, nil
	}

	if len(freeNodes) != len(failedNodes) ||
		(totalNum-len(freeNodes))%(totalRepli-len(freeNodes)) != 0 {
		log.Infof("local", "totalNum=%d totalRepli=%d freeNodes=%d failedNodes=%d",
			totalNum-len(freeNodes), totalRepli-len(freeNodes), len(freeNodes), len(failedNodes))
		return nil, errors.New("cluster fix check error, please check")
	}
	avgReplica := int(totalNum - len(freeNodes)/(totalRepli-len(freeNodes)))
	for _, rs := range rss {
		if len(rs.AllNodes()) < avgReplica {
			defectMaster = append(defectMaster, rs.Master)
		}
	}
	// forget offline nodes
	for _, node := range failedNodes {
		forgetCmd := ForgetAndResetNodeCommand{
			NodeId: node.Id,
		}
		forgetCmd.Execute(c)
		log.Eventf(node.Addr(), "Forget and reset failed node")
	}

	//meet & replicate
	for _, node := range freeNodes {
		meetCmd := MeetNodeCommand{
			NodeId: node.Id,
		}
		meetCmd.Execute(c)
		log.Eventf(node.Addr(), "Meet cluster")
	}

	// give some time to gossip
	time.Sleep(5 * time.Second)

	for idx, node := range freeNodes {
		//disable read
		disableReadCmd := DisableReadCommand{
			NodeId: node.Id,
		}
		disableReadCmd.Execute(c)
		log.Eventf(node.Addr(), "Disable read flag")

		//replicate
		replicateCmd := ReplicateCommand{
			ChildId:  node.Id,
			ParentId: defectMaster[idx].Id,
		}
		replicateCmd.Execute(c)
		log.Eventf(node.Addr(), "Replicate %s to %s", node.Addr(), defectMaster[idx].Addr())
	}

	result := FixClusterResult{Result: true}
	return result, nil
}
