#Redis3.0设计架构
##概述
###背景

社区版本RedisCluster已经加入基于Gossip的集群化方案，各节点通过协议交换配置信息最终达到状态自维护的集群模式。但是由于Ksarch服务的产品线有跨地域需求，而跨地域的网络延迟和网络异常时常发生，进而导致基于Gossip的Cluster内部消息传递和状态判断就变得不可信，跨地域网络故障或网络划分会导致集群状态不可控。

###设计目标
 基于社区RedisCluster实现跨地域情况下的集群管理，可以实现自动failover，实现平台化运维，状态全部收敛于RedisCluster内部，proxy和controller组件全部是无状态，并实现在主从切换时同源增量同步，减小全量同步带来的网络带宽消耗和服务可用性下降。
###名词解释

	Redis3.0：ksarch-redis3.0
	RedisCluster：社区版本Redis3.0
	Region：地域，bj,nj,hz,gz等
	Master Region：主地域，即redis master所在地域
	MachineRoom：物理机房cq01,nj02,nj03...
	LogicMachineRoom：逻辑机房jx,tc,nj,nj03,gz...
	RegionLeader：地域的controller leader
	ClusterLeader：整个集群的controller leader

![architecture](doc/pic/rediscluster1.png)


###系统组件
`RedisCluster`：社区发布Redis3.0.3，增加多IDC支持，支持同源增量同步，支持关闭自动Failover，支持读写权限配置。
`Twemproxy`：基于Twitter的twemproxy开发，增加多地域支持，添加server拓扑自动更新，增加ACK和MOVED协议用于短时间内twemproxy的集群拓扑和rediscluster集群拓扑不一致时的请求转发。
`Controller`：从RedisCluster获取集群状态，然后对集群状态进行判断，对本地域PFAIL节点进行判活，然后将本地域节点的状态发送给ClusterLeader，最终由ClusterLeader进行向集群广播节点FAIL的消息。

###整体架构


单集群多地域架构如图所示。
![arch](doc/pic/rediscluster2.png)

####Controller
Controller工作在地域级别，往往一个地域有多个逻辑机房或物理机房，一个地域可以部署多个controller，但是同一地域内只有一个实际工作。当实际工作的controller挂掉后，集群会进行重新选择新主。
#####功能
	集群控制入口：
	•	节点读写权限控制
	•	节点主从切换
	•	主故障自动选主
	•	从故障自动封禁
	•	数据迁移控制
	•	数据Rebalance

	集群信息：
	•	信息采集
	•	信息处理
	•	cli工具和Dashboard提供集群信息和接口

#####工作原理
Controller在启动时会根据配置中指定的redis seed中的server中随机选择一个来获取集群拓扑，这组seed在生产环境中会配置成本机房内的redis server列表。Controller会定时的进行集群拓扑的获取，随机选择一个seed进行初始创建本地域集群视图，并检查其他seed视角下的集群拓扑是否一致，如果不一致说明集群状态正在变化，需要进行重试。

原生RedisCluster集群中超过半数的节点一致认为不可用节点为PFAIL时会将该不可用节点标记为FAIL，如果该节点是Master，则会向整个集群进行广播节点不可用；如果Slave发现是自己的Master挂掉，则会发起投票选举新主。

修改后的RedisCluster集群中，当出现有的节点不可用后，内部通过Gossip协议会进行节点状态检查，其他节点会标记该不可用的节点状态为PFAIL，该状态为本地视角。并且不会再进行后续的PFAIL到FAIL状态的转换，该部分操作可以配置为由controller自动操作或者需要手动触发完成，RedisCluster集群只用来生成集群拓扑的本地视角。


####Proxy

基于Twemproxy开发，增加集群拓扑发现功能，启动时proxy会从seed列表中随机选择一个server来获取集群状态，然后通过lua脚本解析集群信息并创建拓扑结构和slot信息。

#####拓扑发现
启动后proxy会间隔从RedisCluster通过cluster nodes extra命令来来获取集群信息，这组信息中包含了集群的主从关系和slot的分布，创建请求的路由信息。其中replicaset为一主多从结构，其中多个slave是根据逻辑机房分组。

#####后端failover
•	twemproxy自带failover：
配置文件server_failure_limit配置了后端连续失败几次后在一小段时间内剔除该节点。

•	集群拓扑更新：
       对于请求的每个key，会进行hash取模后对应于16384个slot中的某一个，该slot对应的后端server replicaset，该replicaset是一主多从结构，对于写请求只能请求到Master，对于读请求会根据定义的逻辑机房访问优先顺序从replicaset中选择一个server。当从RedisCluster获取的节点的读写权限为—状态(总共有四种读写权限:rw,r-,-w,—)，则下次创建拓扑时会自动剔除该节点。
       
#####请求过程
由于proxy是间隔从集群内部获取集群拓扑，所以可能存在某时间点proxy获取的集群状态和真正的状态不一致，所以proxy实现了RedisCluster内部的ASK和MOVED命令，当proxy路由信息和集群内部不一致时可以通过请求转发来完成请求。
•	key->slot: hash(key) mod 16384
•	slot->server: 写请求路由到master，读请求按逻辑机房优先顺序选择server


####RedisCluster
##### 多IDC支持
社区原生RedisCluster针对单地域设计，而我们的场景大多数都是一主多从的跨地域部署，Write写主地域，然后同步给各地域的Slave；Read就近访问。

#####同源增量同步
原生社区版本Redis增量同步能力有限，要求主从关系不变，并且连接中断时间有限情况下才可以增量同步。同源情况下切换主，slave也需要进行全量同步。实现是为每个slave都留一个backlog buffer，正常同步过来的写请求都会写backlog buffer和记录与Master断开时间点的LastReplOffset和LastMasterRunid，当需要进行同步时先进行是否可以满足增量同步的条件：
•	新Master的LasterMasterRunid与Slave请求PSYNC时发来的MasterRunid相同
•	新Master的LastReplOffset大于等于Slave请求的Offset
•	LastMasterRunid存在时间小于10秒


