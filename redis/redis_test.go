package redis

import (
	"fmt"
	"testing"
)

func TestIsAlvie(t *testing.T) {
	fmt.Println(IsAlive("127.0.0.1:6379"))
}

func TestClusterNodes(t *testing.T) {
	fmt.Println(ClusterNodes("127.0.0.1:6379"))
}

func TestRedisCli(t *testing.T) {
	reply, _ := RedisCli("127.0.0.1:6379", "config", "get", "maxmemory")
	switch reply.(type) {
	case []string:
		for _, str := range reply.([]string) {
			fmt.Println(str)
		}
	default:
		fmt.Println(reply)
	}
	reply, _ = RedisCli("127.0.0.1:6379", "get", "a")
	switch reply.(type) {
	case []string:
		for _, str := range reply.([]string) {
			fmt.Println(str)
		}
	default:
		fmt.Println(reply)
	}
}
