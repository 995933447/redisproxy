package handler

import (
	"github.com/995933447/redisproxy/redisproxy"
)

type RedisProxy struct {
	redisproxy.UnimplementedRedisProxyServer
	ServiceName string
}

var RedisProxyHandler = &RedisProxy{
	ServiceName: redisproxy.EasymicroGRPCPbServiceNameRedisProxy,
}