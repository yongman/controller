# Redis Docker Demo搭建
如下步骤为在单宿主机上进行搭建demo的步骤.
为了方便执行，先执行如下命令(export一下本机ip地址)

```
export hostip=10.99.201.8
```

## zk搭建

```
//拉取镜像
docker pull hub.c.163.com/zhongjianfeng/zookeeper
//启动服务
docker run -d --name zk --net host hub.c.163.com/zhongjianfeng/zookeeper
```

## 创建redis镜像并启动

cd到redis的[repo](https://github.com/ksarch-saas/redis)目录

```
docker build -t local/redis:v1 .
//启动redis实例

#!/bin/bash

redis_ports_range="2000 2010 2020 2030 2040 2050"

for port in $redis_ports_range
do
	docker run -d --net host --name db_$port local/redis:v1 $port
done
```

## 初始化redis集群命令

```
注：需要自己编译一个redis-cli客户端放到PATH下

#!/bin/bash
set -eu

#meet
for port in 2000 2010 2020 2030 2040 2050
do
	redis-cli -h $hostip -p 2000 cluster meet $hostip $port
done

echo "wait nodes meet each other..."
sleep 5

#replicate
masterid1=`redis-cli -h $hostip -p 2000 cluster nodes | grep myself | awk '{print $1}'`
redis-cli -h $hostip -p 2010 cluster replicate $masterid1

masterid2=`redis-cli -h $hostip -p 2020 cluster nodes | grep myself | awk '{print $1}'`
redis-cli -h $hostip -p 2030 cluster replicate $masterid2

masterid3=`redis-cli -h $hostip -p 2040 cluster nodes | grep myself | awk '{print $1}'`
redis-cli -h $hostip -p 2050 cluster replicate $masterid3

#add slots
redis-cli -h $hostip -p 2000 cluster addslots `seq 0 5461`
redis-cli -h $hostip -p 2020 cluster addslots `seq 5462 10921`
redis-cli -h $hostip -p 2040 cluster addslots `seq 10922 16383`
```

## Proxy镜像并启动

//cd到proxy的[repo](https://github.com/ksarch-saas/r3proxy)根目录，生成镜像

```
docker build -t local/proxy:v1 .

//启动实例
docker run -d --name proxy --net host local/proxy:v1 proxyIP 宿主机IP redis端口
如：
docker run -d --name proxy --net host local/proxy:v1 4000 10.99.201.8 2000
```

## 创建zkcli镜像，初始化zk数据

cd到项目[repo](https://github.com/ksarch-saas/zookeepercli)，创建镜像

```

docker build -t local/zkcli:v1 .

//初始化数据
docker run --rm local/zkcli:v1 init -server $hostip 2181
```

## 生成controller镜像

```
cd <controller_repo>
docker build -t local/controller:v1 .
```

## 创建controller客户端cli，初始化集群数据

[controller_repo](https://github.com/ksarch-saas/cc)

```
cd <controller_repo>/cli
//创建cli镜像
docker build -t local/controllercli:v1 .

//增加集群配置到zk
docker run -it --rm local/controllercli:v1 $hostip:2181 appadd -n redis-demo -r=nj -u=1000000000
```

## 启动controller

[controller_repo](https://github.com/ksarch-saas/cc)

```
docker run -d --name controller --net host local/controller:v1 -appname=redis-demo -http-port=8000 -ws-port=8001 -local-region=nj -seeds=$10.99.201.8:2000 -zkhosts=$hostip:2181
```

## 搭建完成

### 读写测试

```
redis-cli -h $hostip -p 4000
```

### 集群状态查看与控制

```
docker run -it --rm local/controllercli:v1 $hostip:2181 redis-demo

//或者直接通过UI来进行查看和控制
http://$hostip:8000/ui/cluster.html
```
