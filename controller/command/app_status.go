package command

import cc "github.com/ksarch-saas/cc/controller"

type AppStatusCommand struct{}

type AppStatusResult struct {
	Partitions   int
	Replicas     int
	FailedNodes  int
	ReplicaEqual bool
	NodeMax      int64
	NodeMin      int64
	NodeAvg      int64
}

func (self *AppStatusCommand) Execute(c *cc.Controller) (cc.Result, error) {
	cs := c.ClusterState
	snapshot := cs.GetClusterSnapshot()
	if snapshot == nil {
		return nil, nil
	}

	result := &AppStatusResult{
		Partitions:   0,
		Replicas:     0,
		FailedNodes:  0,
		ReplicaEqual: true,
		NodeMax:      0,
		NodeMin:      1024 * 1024 * 1024 * 1024,
		NodeAvg:      0,
	}
	snapshot.BuildReplicaSets()
	nodeState := map[string]string{}
	nss := cs.AllNodeStates()

	for id, ns := range nss {
		nodeState[id] = ns.CurrentState()
	}

	rss := snapshot.ReplicaSets()
	first := true
	var totalMem int64
	for _, rs := range rss {
		if rs.Master != nil && rs.Master.IsArbiter() {
			continue
		}
		if rs.Master != nil {
			usedMem := rs.Master.UsedMemory
			if result.NodeMax < usedMem {
				result.NodeMax = usedMem
			}
			if result.NodeMin > usedMem {
				result.NodeMin = usedMem
			}
			totalMem += usedMem
		}
		if first {
			first = false
			result.Partitions = len(rss)
			result.Replicas = len(rs.AllNodes())
		} else {
			if result.Replicas != len(rs.AllNodes()) {
				result.ReplicaEqual = false
			}
		}
	}
	if len(rss) > 0 {
		result.NodeAvg = totalMem / int64(len(rss))
	}

	for _, ns := range nodeState {
		if ns != "RUNNING" {
			result.FailedNodes++
		}
	}

	return result, nil
}
