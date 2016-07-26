package meta

import (
	"encoding/json"
	"fmt"

	"github.com/golang/glog"
	"github.com/ksarch-saas/cc/topo"
	zookeeper "github.com/samuel/go-zookeeper/zk"
)

type MigrateMeta struct {
	SourceId string
	TargetId string
	Ranges   []topo.Range
	TaskId   string // SourceId[0:6]-TargetId[0:6]
}

func AddMigrateTask(taskMeta *MigrateMeta) error {
	appname := meta.appName
	zconn := meta.zconn
	taskspath := "/r3/app/" + appname + "/migrate"
	exists, _, err := zconn.Exists(taskspath)
	if err != nil {
		return err
	}
	if !exists {
		_, err = zconn.Create(taskspath, []byte(""), 0, zookeeper.WorldACL(zookeeper.PermAll))
		if err != nil {
			return err
		}
	}
	//add task to zk
	taskpath := fmt.Sprintf("%s/%s", taskspath, taskMeta.TaskId)
	exists, _, err = zconn.Exists(taskpath)
	if err != nil {
		return err
	}
	if !exists {
		taskmeta, _ := json.Marshal(taskMeta)
		_, err = zconn.Create(taskpath, taskmeta, 0, zookeeper.WorldACL(zookeeper.PermAll))
		if err != nil {
			return err
		}
	}
	return nil

}

func RemoveMigrateTask(taskId string) error {
	appname := meta.appName
	zconn := meta.zconn
	taskpath := "/r3/app/" + appname + "/migrate/" + taskId
	exists, _, err := zconn.Exists(taskpath)
	if err != nil {
		return err
	}
	if exists {
		err = zconn.Delete(taskpath, -1)
		if err != nil {
			return err
		}
	}
	return nil
}

func AllMigrateTasks() ([]*MigrateMeta, error) {
	appname := meta.appName
	zconn := meta.zconn
	taskspath := "/r3/app/" + appname + "/migrate"
	exists, _, err := zconn.Exists(taskspath)
	if err != nil {
		return nil, err
	}
	if exists {
		tasks, _, err := zconn.Children(taskspath)
		if err != nil {
			return nil, err
		}
		var alltasks []*MigrateMeta
		for _, t := range tasks {
			glog.Info("Get task from zk: ", t)
			task, _, err := zconn.Get(taskspath + "/" + t)
			if err != nil {
				glog.Warning("Get task failed ", err)
				return nil, err
			}
			var migrateMeta MigrateMeta
			err = json.Unmarshal(task, &migrateMeta)
			if err != nil {
				glog.Warning("Cluster", "Unmarshal failed: ", err)
				return nil, err
			}
			alltasks = append(alltasks, &migrateMeta)
		}
		return alltasks, nil
	} else {
		return nil, nil
	}
}
