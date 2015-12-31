package topo

import "errors"

var (
	ErrInvalidParentId = errors.New("topo: invalid parent id, master not exist")
)

type FailureInfo struct {
	Seeds []*Node
}

type ClusterInfo struct {
	ClusterState                 string
	ClusterSlotsAssigned         int
	ClusterSlotsOk               int
	ClusterSlotsPfail            int
	ClusterSlotsFail             int
	ClusterKnownNodes            int
	ClusterSize                  int
	ClusterCurrentEpoch          int
	ClusterMyEpoch               int
	ClusterStatsMessagesSent     int
	ClusterStatsMessagesReceived int
}

type Cluster struct {
	localRegion      string
	localRegionNodes []*Node
	nodes            []*Node
	replicaSets      []*ReplicaSet
	idTable          map[string]*Node
}

func NewCluster(region string) *Cluster {
	c := &Cluster{}
	c.localRegion = region
	c.localRegionNodes = []*Node{}
	c.nodes = []*Node{}
	c.replicaSets = []*ReplicaSet{}
	c.idTable = map[string]*Node{}
	return c
}

func (self *Cluster) AddNode(s *Node) {
	self.idTable[s.Id] = s
	self.nodes = append(self.nodes, s)

	if s.Region == self.localRegion {
		self.localRegionNodes = append(self.localRegionNodes, s)
	}
}

func (self *Cluster) AllNodes() []*Node {
	return self.nodes
}

func (self *Cluster) NumNode() int {
	return len(self.nodes)
}

func (self *Cluster) LocalRegionNodes() []*Node {
	return self.localRegionNodes
}

func (self *Cluster) MasterNodes() []*Node {
	nodes := []*Node{}
	for _, node := range self.AllNodes() {
		if node.IsMaster() {
			nodes = append(nodes, node)
		}
	}
	return nodes
}

func (self *Cluster) Size() int {
	return len(self.MasterNodes())
}

func (self *Cluster) RegionNodes(region string) []*Node {
	nodes := []*Node{}
	for _, n := range self.nodes {
		if n.Region == region {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

func (self *Cluster) NumLocalRegionNode() int {
	return len(self.localRegionNodes)
}

func (self *Cluster) FindNode(id string) *Node {
	return self.idTable[id]
}

func (self *Cluster) FindNodeBySlot(slot int) *Node {
	for _, node := range self.AllNodes() {
		if node.IsMaster() {
			for _, r := range node.Ranges {
				if slot >= r.Left && slot <= r.Right {
					return node
				}
			}
		}
	}
	return nil
}

func (self *Cluster) FindReplicaSetByNode(id string) *ReplicaSet {
	for _, rs := range self.replicaSets {
		if rs.HasNode(id) {
			return rs
		}
	}
	return nil
}

func (self *Cluster) Region() string {
	return self.localRegion
}

func (self *Cluster) FailureNodes() []*Node {
	ss := []*Node{}
	for _, s := range self.localRegionNodes {
		if s.Fail {
			ss = append(ss, s)
		}
	}
	return ss
}

func (self *Cluster) BuildReplicaSets() error {
	replicaSets := []*ReplicaSet{}

	for _, s := range self.nodes {
		if s.IsMaster() {
			rs := NewReplicaSet()
			rs.SetMaster(s)
			replicaSets = append(replicaSets, rs)
		}
	}

	for _, s := range self.nodes {
		if !s.IsMaster() {
			master := self.FindNode(s.ParentId)
			if master == nil {
				return ErrInvalidParentId
			}

			for _, rs := range replicaSets {
				if rs.Master == master {
					rs.AddSlave(s)
				}
			}
		}
	}

	self.replicaSets = replicaSets
	return nil
}

type ByMasterId []*ReplicaSet
type ByNodeState []*ReplicaSet

// sort by master id
func (a ByMasterId) Len() int           { return len(a) }
func (a ByMasterId) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByMasterId) Less(i, j int) bool { return a[i].Master.Id < a[j].Master.Id }

// sort replicas by node state
func (a ByNodeState) Len() int      { return len(a) }
func (a ByNodeState) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByNodeState) Less(i, j int) bool {
	//inner function to check a relicas is normal
	inner := func(idx int) bool {
		if a[idx].Master.PFail || a[idx].Master.Fail || a[idx].Master.Free ||
			(!a[idx].Master.Readable && !a[idx].Master.Writable) {
			return true
		}
		for _, node := range a[idx].Slaves {
			if node.PFail || node.Fail || node.Free ||
				(!node.Readable && !node.Writable) {
				return true
			}
		}
		return false
	}
	if len(a[i].Slaves) < len(a[j].Slaves) {
		return false
	}
	if inner(i) && !inner(j) {
		return false
	}
	return true
}

func (self *Cluster) ReplicaSets() []*ReplicaSet {
	return self.replicaSets
}

func (self *Cluster) NumReplicaSets() int {
	return len(self.replicaSets)
}

func (self *Cluster) String() string {
	return ""
}
