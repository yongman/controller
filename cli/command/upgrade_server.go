package command

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/codegangsta/cli"

	"github.com/ksarch-saas/cc/cli/context"
	"github.com/ksarch-saas/cc/controller/command"
	"github.com/ksarch-saas/cc/frontend/api"
	"github.com/ksarch-saas/cc/redis"
	"github.com/ksarch-saas/cc/topo"
	"github.com/ksarch-saas/cc/utils"
)

var UpgradeServerCommand = cli.Command{
	Name:   "upgradeserver",
	Usage:  "upgradeserver -r=<slaves|master>",
	Action: upgradeServerAction,
	Flags: []cli.Flag{
		cli.StringFlag{"r,role", "", "upgrade slaves or master server"},
	},
	Description: `
	upgrade redis server
	`,
}
var IdxServerAddr = "ksarch-saas.baidu.com:7379"

// 0 upgrade program env
// 1 disable autofailover
// 2 disable autoenableslave read

// steps slaves upgrade
// 1 disable all slaves read flag
// 2 disable aof feature and rdb feature
// 3 restart all slaves server(send 'shutdown nosave' command, supervisord will restart the server)
// 4 check the status of slaves server
// 5 save the idx of the process
// 6 enable aof all of the slaves
// 7 next replicaSet... and again

// steps master upgrade
// 1 manualfailover to master
// 2 disable read flag
// 3 check failover status done
// 3 restart the old master node(shutdown nosave)
// 4 check the status of slaves server
// 5 save the idx of the process
// 6 enable the read flag
// 7 next replicaSet… and again

func configRead(node *topo.Node, state bool) (*api.Response, error) {
	addr := context.GetLeaderAddr()
	//send failover command to new_master
	extraHeader := &utils.ExtraHeader{
		User:  context.Config.User,
		Role:  context.Config.Role,
		Token: context.Config.Token,
	}

	url_perm := "http://" + addr + api.NodePermPath
	req_disableread := api.ToggleModeParams{
		NodeId: node.Id,
		Action: "disable",
		Perm:   "read",
	}
	req_enableread := api.ToggleModeParams{
		NodeId: node.Id,
		Action: "enable",
		Perm:   "read",
	}
	var req api.ToggleModeParams
	if state {
		req = req_enableread
	} else {
		req = req_disableread
	}
	resp, err := utils.HttpPostExtra(url_perm, req, 5*time.Second, extraHeader)
	return resp, err
}

func configAofAndRdb(node *topo.Node, state bool) error {
	addr := node.Addr()
	var err error
	var err1 error
	var err2 error
	if state {
		_, err = redis.RedisCli(addr, "config", "set", "appendonly", "yes")
		_, err1 = redis.RedisCli(addr, "config", "set", "dbfilename", "dump.rdb")
	} else {
		_, err = redis.RedisCli(addr, "config", "set", "appendonly", "no")
		_, err1 = redis.RedisCli(addr, "config", "set", "dbfilename", "tmp.rdb")
	}
	_, err2 = redis.RedisCli(addr, "config", "rewrite")
	if err != nil {
		return err
	}
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

func checkSlaveRepliStatusOk(node *topo.Node) (bool, error) {
	addr := node.Addr()
	info, err := redis.FetchInfo(addr, "all")
	if err != nil {
		return false, err
	}
	if info.Get("role") == "master" {
		return false, nil
	}
	if info.Get("master_link_status") == "down" {
		return false, nil
	} else if info.Get("loading") == "1" {
		return false, nil
	} else {
		return true, nil
	}
}

func checkMasterRole(node *topo.Node, ismaster bool) (bool, error) {
	addr := node.Addr()
	info, err := redis.FetchInfo(addr, "replication")
	if err != nil {
		return false, fmt.Errorf("Connect %s failed", addr)
	}
	if info.Get("role") == "master" {
		return true, nil
	} else {
		return false, nil
	}
}

func shutdownServer(node *topo.Node) error {
	addr := node.Addr()
	_, err := redis.RedisCli(addr, "shutdown", "nosave")
	return err
}

func saveIdx(addr string, pid string, role string, replica_idx int) error {
	key := fmt.Sprintf("%s_%s_%s", "upgrade_index", pid, role)
	_, err := redis.RedisCli(addr, "set", key, replica_idx)
	return err
}

func getIdx(addr string, pid string, role string) (int, error) {
	key := fmt.Sprintf("%s_%s_%s", "upgrade_index", pid, role)
	idx, err := redis.RedisCli(addr, "get", key)
	if idx == "" {
		return -1, nil
	}
	iidx, _ := strconv.Atoi(idx.(string))
	return iidx, err
}

func upgradeServerAction(c *cli.Context) {
	role := c.String("r")
	if role == "" {
		fmt.Println(ErrInvalidParameter)
		return
	}
	if role == "slaves" {
		upgradeSlaves(c)
	} else if role == "master" {
		upgradeMaster(c)
	} else {
		fmt.Println("Err: role must be slaves or master")
	}
}

func upgradeMaster(c *cli.Context) {
	pid := context.GetAppName()
	addr := context.GetLeaderAddr()
	url_fr := "http://" + addr + api.FetchReplicaSetsPath
	url_fl := "http://" + addr + api.NodeSetAsMasterPath
	extraHeader := &utils.ExtraHeader{
		User:  context.Config.User,
		Role:  context.Config.Role,
		Token: context.Config.Token,
	}

	resp, err := utils.HttpGet(url_fr, nil, 5*time.Second)
	if err != nil {
		fmt.Println(err)
		return
	}

	var rss command.FetchReplicaSetsResult
	err = utils.InterfaceToStruct(resp.Body, &rss)
	if err != nil {
		fmt.Println(err)
		return
	}
	sort.Sort(topo.ByMasterId(rss.ReplicaSets))
	sort.Sort(topo.ByNodeState(rss.ReplicaSets))

	iidx, err := getIdx(IdxServerAddr, pid, "master")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Get last idx record: %d\n", iidx)
	var old_master *topo.Node
	var new_master *topo.Node
	old_master = nil
	new_master = nil

	for idx, rs := range rss.ReplicaSets {
		if idx <= iidx {
			fmt.Printf("Skipping replica(id:%s) (%d/%d) master\n", rs.Master.Id, idx, len(rss.ReplicaSets))
			continue
		}
		//select a slave in the same IDC
		old_master = rs.Master
		old_master_r := getRegion(old_master)
		if old_master_r == "" {
			return
		}

		fmt.Printf("Upgrading replica(id:%s) (%d/%d) master\n", rs.Master.Id, idx, len(rss.ReplicaSets))
		for _, s := range rs.Slaves {
			re := getRegion(s)
			if re == "" {
				return
			}
			if re == old_master_r {
				new_master = s
				break
			}
		}
		if new_master == nil {
			fmt.Printf("Select new master failed for master(%s) replica\n", old_master.Id)
			return
		}
		//send failover to the new master
		req := api.FailoverTakeoverParams{
			NodeId: new_master.Id,
		}
		resp, err := utils.HttpPostExtra(url_fl, req, 10*time.Second, extraHeader)
		if err != nil {
			fmt.Println(err)
			return
		}
		if resp.Errno != 0 {
			fmt.Println(resp.Errmsg)
			return
		}
		//send failover request done,check the new_master role to a real master
		for {
			ismaster, err := checkMasterRole(new_master, true)
			if err != nil {
				fmt.Println(err)
				time.Sleep(10 * time.Second)
				continue
			}
			if ismaster == true {
				//to be a new master
				break
			} else {
				//wait for next check
				time.Sleep(10 * time.Second)
			}
		}
		//disable read flag of the old_master
		resp, err = configRead(old_master, false)
		if err != nil {
			fmt.Println(err)
			return
		}
		if resp.Errno != 0 {
			fmt.Println(resp.Errmsg)
			return
		}
		//disable aof and rdb to speed up start
		err = configAofAndRdb(old_master, false)
		if err != nil {
			fmt.Println(err)
			return
		}
		//shutdown server
		err = shutdownServer(old_master)
		if err != nil {
			fmt.Printf("server %s restart\n", old_master.Addr())
		}
		//check the status of old master
		cnt := 1
		for {
			fmt.Printf("Check slave status %d times\n", cnt)
			cnt++

			ok, err := checkSlaveRepliStatusOk(old_master)
			if err != nil || !ok {
				//not ok, wait for next trun check
				time.Sleep(10 * time.Second)
			} else {
				break
			}
		}
		//enable aof and rdb
		err = configAofAndRdb(old_master, true)
		if err != nil {
			fmt.Println(err)
			return
		}
		//enable read flag of the old_master
		resp, err = configRead(old_master, true)
		if err != nil {
			fmt.Println(err)
			return
		}
		if resp.Errno != 0 {
			fmt.Println(resp.Errmsg)
			return
		}
		//save the idx of the process
		err = saveIdx(IdxServerAddr, pid, "master", idx)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

func upgradeSlaves(c *cli.Context) {

	pid := context.GetAppName()
	addr := context.GetLeaderAddr()
	url := "http://" + addr + api.FetchReplicaSetsPath

	resp, err := utils.HttpGet(url, nil, 5*time.Second)
	if err != nil {
		fmt.Println(err)
		return
	}

	var rss command.FetchReplicaSetsResult
	err = utils.InterfaceToStruct(resp.Body, &rss)
	if err != nil {
		fmt.Println(err)
		return
	}
	sort.Sort(topo.ByMasterId(rss.ReplicaSets))
	sort.Sort(topo.ByNodeState(rss.ReplicaSets))

	iidx, err := getIdx(IdxServerAddr, pid, "slaves")
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Get last idx record: %d\n", iidx)
	for idx, rs := range rss.ReplicaSets {
		if idx <= iidx {
			fmt.Printf("Skipping replica(id:%s) (%d/%d) slaves\n", rs.Master.Id, idx, len(rss.ReplicaSets))
			continue
		}

		fmt.Printf("Upgrading replica(id:%s) (%d/%d) slaves\n", rs.Master.Id, idx, len(rss.ReplicaSets))
		for _, s := range rs.Slaves {

			//disable read
			_, err := configRead(s, false)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("Disable read %s\n", s.Addr())

			err = configAofAndRdb(s, false)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("Disable aof feature %s\n", s.Addr())

			//send shutdown command
			err = shutdownServer(s)
			if err != nil {
				fmt.Printf("server %s restart\n", s.Addr())
			}
			//sleep for 5 seconds
			time.Sleep(5 * time.Second)
		}
		//check slaves replica status and loading status
		inner := func(nodes []*topo.Node) bool {
			for _, n := range nodes {
				ok, err := checkSlaveRepliStatusOk(n)
				if err != nil {
					return false
				}
				if !ok {
					return false
				}
			}
			return true
		}
		cnt := 0
		for {
			ok := inner(rs.Slaves)
			if ok {
				break
			}
			cnt++
			fmt.Printf("Checking slaves replication status %d times\n", cnt)
			time.Sleep(5 * time.Second)
		}
		//enable slaves aof and read flag
		for _, s := range rs.Slaves {
			err := configAofAndRdb(s, true)
			if err != nil {
				fmt.Println(err)
			}
			_, err = configRead(s, true)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("Enable slaves %s aof and read flag\n", s.Addr())
		}
		//status ok, record the idx to a redis
		err := saveIdx(IdxServerAddr, pid, "slaves", idx)
		if err != nil {
			fmt.Printf("saveIdx to %d failed\n", idx)
		}
	}
}
