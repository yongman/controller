package inspector

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/ksarch-saas/cc/frontend/api"
	"github.com/ksarch-saas/cc/meta"
	"github.com/ksarch-saas/cc/topo"
	"github.com/ksarch-saas/cc/utils"
)

func MkUrl(path string) string {
	return "http://" + meta.LeaderHttpAddress() + path
}

func SendRegionTopoSnapshot(nodes []*topo.Node, failureInfo *topo.FailureInfo) error {
	params := &api.RegionSnapshotParams{
		Region:      meta.LocalRegion(),
		PostTime:    time.Now().Unix(),
		Nodes:       nodes,
		FailureInfo: failureInfo,
	}

	resp, err := utils.HttpPost(MkUrl(api.RegionSnapshotPath), params, 30*time.Second)
	if err != nil {
		return err
	}
	if resp.Errno != 0 {
		return fmt.Errorf("%d %s", resp.Errno, resp.Errmsg)
	}
	return nil
}

func containsNode(node *topo.Node, nodes []*topo.Node) bool {
	for _, n := range nodes {
		if n.Id == node.Id {
			return true
		}
	}
	return false
}

func (self *Inspector) IsClusterDamaged(cluster *topo.Cluster, seeds []*topo.Node) bool {
	// more than half masters dead
	numFail := 0
	for _, node := range cluster.MasterNodes() {
		if node.Fail {
			numFail++
		}
	}
	if numFail >= (cluster.Size()+1)/2 {
		return true
	}

	// more than half nodes dead
	if len(seeds) > cluster.NumLocalRegionNode()/2 {
		return false
	}
	for _, seed := range seeds {
		c, err := self.initClusterTopo(seed)
		if err != nil {
			return false
		}
		for _, node := range c.LocalRegionNodes() {
			// nodes not in seeds must be pfail
			if !containsNode(node, seeds) && !node.PFail {
				return false
			}
		}
	}
	glog.Info("more than half nodes dead")
	return true
}

func FixClusterCircle() {
	// 定时fixcluster，针对 len(failnodes) == len (freednodes) 情况自动修复
	app := meta.GetAppConfig()
	trickerTime := app.FixClusterCircle
	aotuFixCluster := app.AotuFixCluster

	if trickerTime == 0 {
		trickerTime = meta.DEFAULT_FIXCLUSTER_CIRCLE
	}
	tickChan := time.NewTicker(time.Second * time.Duration(trickerTime)).C
	for {
		select {
			case <-tickChan:
				glog.Infof("ClusterLeader:%s, RegionLeader:%s", meta.ClusterLeaderZNodeName(), meta.RegionLeaderZNodeName())
				if meta.IsClusterLeader() && aotuFixCluster{				
					addr := meta.LeaderHttpAddress()
					url := "http://" + addr + api.FixClusterPath
					_, err := utils.HttpPost(url, nil, 0)
					if err != nil {
						glog.Info(err.Error())
					}
				}
		}
	}
}

func (self *Inspector) Run() {
	go FixClusterCircle()
	appconfig := meta.GetAppConfig()
	// FetchClusterNodesInterval not support heat loading
	tickChan := time.NewTicker(appconfig.FetchClusterNodesInterval).C
	for {
		select {
		case <-tickChan:
			if !meta.IsRegionLeader() {
				continue
			}
			cluster, seeds, err := self.BuildClusterTopo()
			if err != nil {
				glog.Infof("build cluster topo failed, %v", err)
			}
			if cluster == nil {
				continue
			}
			var failureInfo *topo.FailureInfo
			if meta.IsInMasterRegion() && self.IsClusterDamaged(cluster, seeds) {
				failureInfo = &topo.FailureInfo{Seeds: seeds}
			}
			var nodes []*topo.Node
			if err == nil {
				nodes = cluster.LocalRegionNodes()
			}
			err = SendRegionTopoSnapshot(nodes, failureInfo)
			if err != nil {
				glog.Infof("send snapshot failed, %v", err)
			}
		}
	}
}
