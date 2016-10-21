package command

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/codegangsta/cli"
	"github.com/ksarch-saas/cc/cli/context"
	"github.com/ksarch-saas/cc/frontend/api"
	"github.com/ksarch-saas/cc/topo"
	"github.com/ksarch-saas/cc/utils"
)

var RebalanceCommand = cli.Command{
	Name:   "rebalance",
	Usage:  "rebalance",
	Action: rebalanceAction,
	Flags: []cli.Flag{
		cli.StringFlag{"m,method", "default", "rebalance methed: default|cuttail|mergetail|mergeall"},
		cli.IntFlag{"c,ratio", 2, "used with merge rebalance method"},
		cli.BoolFlag{"r,run", "run or show rebalance plans only"},
	},
}

func rebalanceAction(c *cli.Context) {
	if len(c.Args()) != 0 {
		fmt.Println(ErrInvalidParameter)
		return
	}
	addr := context.GetLeaderAddr()

	extraHeader := &utils.ExtraHeader{
		User:  context.Config.User,
		Role:  context.Config.Role,
		Token: context.Config.Token,
	}

	url := "http://" + addr + api.RebalancePath

	method := c.String("m")
	ratio := c.Int("c")
	show := c.Bool("r") == false
	req := api.RebalanceParams{
		Method:       method,
		Ratio:        ratio,
		ShowPlanOnly: show,
	}
	resp, err := utils.HttpPostExtra(url, req, 5*time.Second, extraHeader)
	if err != nil {
		fmt.Println(err)
		return
	}
	for _, v := range resp.Body.([]interface{}) {
		sourceId := v.(map[string]interface{})["SourceId"]
		targetId := v.(map[string]interface{})["TargetId"]
		ranges := v.(map[string]interface{})["Ranges"]
		var rangesT topo.Ranges
		for _, r := range ranges.([]interface{}) {
			ri := r.(map[string]interface{})
			left, _ := ri["Left"].(json.Number).Int64()
			right, _ := ri["Right"].(json.Number).Int64()
			rs := topo.Range{
				Left:  int(left),
				Right: int(right),
			}
			rangesT = append(rangesT, rs)
		}
		plan := fmt.Sprintf("%s => %s Ranges:%s", sourceId.(string), targetId.(string), rangesT.String())
		fmt.Println(plan)
	}
}
