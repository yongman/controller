package migrate

import (
	"math"

	"github.com/ksarch-saas/cc/topo"
)

func ManyToOne(sNodes []*topo.Node, tNode *topo.Node) (plans []*MigratePlan) {
	ratio := len(sNodes) + 1
	for _, node := range sNodes {
		parts := node.RangesSplitN(ratio)
		plan := &MigratePlan{
			SourceId: node.Id,
			TargetId: tNode.Id,
			Ranges:   parts[0],
		}
		plans = append(plans, plan)
	}
	return plans
}

func OneToMany(sNode *topo.Node, tNodes []*topo.Node) (plans []*MigratePlan) {
	ratio := len(tNodes) + 1
	parts := sNode.RangesSplitN(ratio)
	for i, node := range tNodes {
		plan := &MigratePlan{
			SourceId: sNode.Id,
			TargetId: node.Id,
			Ranges:   parts[i],
		}
		plans = append(plans, plan)
	}
	return plans
}

func CutTailRebalancer(ss []*topo.Node, ts []*topo.Node) (plans []*MigratePlan) {
	var i int

	numSource := len(ss)
	numTarget := len(ts)

	if numSource == 0 || numTarget == 0 {
		return
	}

	// [s] [s] [s] | [t] [t]
	if numSource >= numTarget {
		ratio := int(math.Ceil(float64(numSource) / float64(numTarget)))
		for i = 0; i < len(ts)-1; i++ {
			tNode := ts[i]
			sNodes := ss[i*ratio : (i+1)*ratio]
			subPlans := ManyToOne(sNodes, tNode)
			plans = append(plans, subPlans...)
		}
		tNode := ts[i]
		sNodes := ss[i*ratio:]
		subPlans := ManyToOne(sNodes, tNode)
		plans = append(plans, subPlans...)
	}

	// [s] [s] | [t] [t] [t]
	if numSource < numTarget {
		ratio := int(math.Ceil(float64(numTarget) / float64(numSource)))
		for i = 0; i < len(ss)-1; i++ {
			sNode := ss[i]
			tNodes := ts[i*ratio : (i+1)*ratio]
			subPlans := OneToMany(sNode, tNodes)
			plans = append(plans, subPlans...)
		}
		sNode := ss[i]
		tNodes := ts[i*ratio:]
		subPlans := OneToMany(sNode, tNodes)
		plans = append(plans, subPlans...)
	}

	return plans
}

func MergerRebalancer(ss []*topo.Node, ts []*topo.Node) (plans []*MigratePlan) {
	//ts should be nil
	//we calculate merger scheme accroding the current slot's distribution
	nodesCount := len(ss)
	if nodesCount == 0 {
		return nil
	}

	ps := -1
	pt := 0
	for _, s := range ss {
		if ps < len(s.Ranges) {
			ps = len(s.Ranges)
		}
	}
	ps = ps + 1
	pt = ps - 1

	for _, s := range ss {
		if len(s.Ranges) == pt {
			//this slots should be split to merge their buddy
			for _, r := range s.Ranges {
				//find the range's buddy
				for _, si := range ss {
					if s == si {
						continue
					}
					for _, ri := range si.Ranges {
						if r.Left == ri.Right+1 {
							//buddy found
							//generate a plan
							plan := &MigratePlan{
								SourceId: s.Id,
								TargetId: si.Id,
								Ranges:   []topo.Range{r},
							}
							plans = append(plans, plan)
						}
					}
				}
			}
		}
	}
	return plans
}
