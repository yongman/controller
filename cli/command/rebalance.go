package command

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/codegangsta/cli"
	"github.com/ksarch-saas/cc/cli/context"
	"github.com/ksarch-saas/cc/frontend/api"
	"github.com/ksarch-saas/cc/utils"
)

var RebalanceCommand = cli.Command{
	Name:   "rebalance",
	Usage:  "rebalance",
	Action: rebalanceAction,
	Flags: []cli.Flag{
		cli.StringFlag{"m,method", "default", "rebalance methed: default or cuttail"},
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
	show := c.Bool("r") == false
	req := api.RebalanceParams{
		Method:       method,
		ShowPlanOnly: show,
	}
	resp, err := utils.HttpPostExtra(url, req, 5*time.Second, extraHeader)
	if err != nil {
		fmt.Println(err)
		return
	}
	//ShowResponse(resp)
	res, err := json.MarshalIndent(resp.Body, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(string(res))
}
