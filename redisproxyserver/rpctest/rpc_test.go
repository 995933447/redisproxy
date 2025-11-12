package rpctest

import (
	"context"
	"log"
	"testing"

	"github.com/995933447/fastlog"
	"github.com/995933447/redisproxy/redisproxy"
	"github.com/995933447/redisproxy/redisproxyserver/boot"
	"github.com/995933447/redisproxy/redisproxyserver/config"
	"github.com/995933447/runtimeutil"
	"github.com/gomodule/redigo/redis"
)

func TestRPC(t *testing.T) {
	InitEnv()
	_, err := redisproxy.DoCmd(context.TODO(), false, 3600, "default", "SET", "hello", "world")
	if err != nil {
		fastlog.Errorf("redis send failed, err:%v", err)
		t.Fatal(err)
	}
	value, err := redis.String(redisproxy.DoCmd(context.TODO(), false, 0, "default", "GET", "hello"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(value)
	_, err = redisproxy.DoCmd(context.TODO(), false, 3600, "default", "HSET", "hello_m", "hello", "world", "foo", "bar")
	if err != nil {
		fastlog.Errorf("redis send failed, err:%v", err)
		t.Fatal(err)
	}
	hValue, err := redis.Strings(redisproxy.DoCmd(context.TODO(), false, 0, "default", "HGETALL", "hello_m"))
	if err != nil {
		t.Fatal(err)
	}
	t.Log(hValue)
}

func InitEnv() {
	if err := boot.InitNode("redisproxy_rpc_test"); err != nil {
		log.Fatal(runtimeutil.NewStackErr(err))
	}

	if err := config.LoadConfig(); err != nil {
		log.Fatal(runtimeutil.NewStackErr(err))
	}

	boot.InitRouteredis()

	var discoveryName string
	config.SafeReadServerConfig(func(c *config.ServerConfig) {
		discoveryName = c.GetDiscoveryName()
	})

	err := redisproxy.PrepareGRPC(discoveryName)
	if err != nil {
		log.Fatal(runtimeutil.NewStackErr(err))
	}
}
