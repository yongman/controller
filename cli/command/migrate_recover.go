package command

import (
	"fmt"
	"time"

	"github.com/codegangsta/cli"
	"github.com/ksarch-saas/cc/cli/context"
	"github.com/ksarch-saas/cc/frontend/api"
	"github.com/ksarch-saas/cc/utils"
)

var MigrateRecoverCommand = cli.Command{
	Name:   "migraterecover",
	Usage:  "migraterecover",
	Action: migrateRecoverAction,
	Flags: []cli.Flag{
		cli.BoolFlag{"r,run", "run or show rebalance plans only"},
	},
}

func migrateRecoverAction(c *cli.Context) {
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

	url := "http://" + addr + api.MigrateRecoverPath

	show := c.Bool("r") == false
	req := api.MigrateRecoverParams{
		ShowOnly: show,
	}
	resp, err := utils.HttpPostExtra(url, req, 5*time.Second, extraHeader)
	if err != nil {
		fmt.Println(err)
		return
	}
	ShowResponseArray(resp)
}
