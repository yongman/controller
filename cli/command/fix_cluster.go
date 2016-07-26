package command

import (
	"github.com/codegangsta/cli"

	"github.com/ksarch-saas/cc/cli/context"
	"github.com/ksarch-saas/cc/frontend/api"
	"github.com/ksarch-saas/cc/utils"
)

var FixClusterCommand = cli.Command{
	Name:   "fixcluster",
	Usage:  "fix cluster",
	Action: fixClusterAction,
	Description: `
	fix cluster, forget failed nodes and meet new nodes
	`,
}

func fixClusterAction(c *cli.Context) {
	addr := context.GetLeaderAddr()

	url := "http://" + addr + api.FixClusterPath

	resp, err := utils.HttpPost(url, nil, 0)
	if err != nil {
		Put(err)
		return
	}
	if resp != nil {
		ShowResponse(resp)
	} else {
		Put("Nil response")
	}
}
