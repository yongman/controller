package command

import (
	cc "github.com/jxwr/cc/controller"
	"github.com/jxwr/cc/redis"
)

type SetAsMasterCommand struct {
	NodeId string
}

func (self *SetAsMasterCommand) Execute(c *cc.Controller) (cc.Result, error) {
	cs := c.ClusterState
	node := cs.FindNode(self.NodeId)
	if node == nil {
		return nil, ErrNodeNotExist
	}
	if node.IsMaster() {
		return nil, ErrNodeIsMaster
	}
	_, err := redis.ClusterFailover(node.Addr())
	if err != nil {
		return nil, err
	}
	for _, ns := range cs.AllNodeStates() {
		_, err = redis.EnableWrite(ns.Addr(), self.NodeId)
		if err == nil {
			return nil, nil
		}
	}
	return nil, err
}
