package command

import (
	cc "github.com/ksarch-saas/cc/controller"
)

/// Command types
func (self *EnableReadCommand) Mutex() cc.CommandType          { return cc.NOMUTEX_COMMAND }
func (self *DisableReadCommand) Mutex() cc.CommandType         { return cc.NOMUTEX_COMMAND }
func (self *EnableWriteCommand) Mutex() cc.CommandType         { return cc.NOMUTEX_COMMAND }
func (self *DisableWriteCommand) Mutex() cc.CommandType        { return cc.NOMUTEX_COMMAND }
func (self *MakeReplicaSetCommand) Mutex() cc.CommandType      { return cc.NOMUTEX_COMMAND }
func (self *ForgetAndResetNodeCommand) Mutex() cc.CommandType  { return cc.NOMUTEX_COMMAND }
func (self *FailoverBeginCommand) Mutex() cc.CommandType       { return cc.NOMUTEX_COMMAND }
func (self *FetchReplicaSetsCommand) Mutex() cc.CommandType    { return cc.NOMUTEX_COMMAND }
func (self *FailoverTakeoverCommand) Mutex() cc.CommandType    { return cc.NOMUTEX_COMMAND }
func (self *MeetNodeCommand) Mutex() cc.CommandType            { return cc.NOMUTEX_COMMAND }
func (self *ReplicateCommand) Mutex() cc.CommandType           { return cc.NOMUTEX_COMMAND }
func (self *MigrateCommand) Mutex() cc.CommandType             { return cc.MUTEX_COMMAND }
func (self *MigratePauseCommand) Mutex() cc.CommandType        { return cc.MUTEX_COMMAND }
func (self *MigrateResumeCommand) Mutex() cc.CommandType       { return cc.MUTEX_COMMAND }
func (self *MigrateCancelCommand) Mutex() cc.CommandType       { return cc.MUTEX_COMMAND }
func (self *SetAsMasterCommand) Mutex() cc.CommandType         { return cc.NOMUTEX_COMMAND }
func (self *UpdateRegionCommand) Mutex() cc.CommandType        { return cc.MUTEX_COMMAND }
func (self *RebalanceCommand) Mutex() cc.CommandType           { return cc.MUTEX_COMMAND }
func (self *FetchMigrationTasksCommand) Mutex() cc.CommandType { return cc.NOMUTEX_COMMAND }
func (self *MergeSeedsCommand) Mutex() cc.CommandType          { return cc.MUTEX_COMMAND }
func (self *MigrateRecoverCommand) Mutex() cc.CommandType      { return cc.MUTEX_COMMAND }
func (self *FixClusterCommand) Mutex() cc.CommandType          { return cc.MUTEX_COMMAND }
