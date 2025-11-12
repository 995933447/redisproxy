package handler

import (
	"context"
	"errors"

	"github.com/995933447/easymicro/grpc"
	"github.com/995933447/fastlog"
	"github.com/995933447/redisproxy/redisproxy"
	"github.com/995933447/routeredis"
	"github.com/gomodule/redigo/redis"
)

func (s *RedisProxy) Do(ctx context.Context, req *redisproxy.DoReq) (*redisproxy.DoResp, error) {
	var resp redisproxy.DoResp

	conn, err := routeredis.GetConn(req.Key.Conn)
	if err != nil {
		fastlog.Errorf("get redis conn failed, err:%v", err)
		return nil, err
	}

	if err = conn.Err(); err != nil {
		fastlog.Errorf("get redis conn failed, err:%v", conn.Err())
		return nil, err
	}

	defer conn.Close()

	args, isMergedTtl, isMergedTtlSetnxCmd := s.buildArgs(req)

	if req.IsAsync {
		err = conn.Send(req.Command, args...)
		if err != nil {
			if errors.Is(err, redis.ErrNil) {
				return nil, grpc.NewRPCErr(redisproxy.ErrCode_ErrCodeRedisNil)
			}
			fastlog.Errorf("redis send failed, err:%v", err)
			return nil, err
		}
	} else {
		reply, err := conn.Do(req.Command, args...)
		if err != nil {
			if errors.Is(err, redis.ErrNil) {
				return nil, grpc.NewRPCErr(redisproxy.ErrCode_ErrCodeRedisNil)
			}
			fastlog.Errorf("redis do failed, err:%v", err)
			return nil, err
		}
		s.decodeReply(reply, &resp)
		if isMergedTtlSetnxCmd {
			resp.Type = int32(redisproxy.Type_TypeInt64)
			if resp.StrValue == "OK" {
				resp.Int64Value = 1
			}
		}
	}

	if !isMergedTtl && req.Ttl > 0 {
		if !req.IsAsync {
			_, err = conn.Do("EXPIRE", req.Key.Key, req.Ttl)
			if err != nil {
				if !errors.Is(err, redis.ErrNil) {
					fastlog.Errorf("redis EXPIRE failed, err:%v", err)
					return nil, err
				}
			}
		} else {
			err = conn.Send("EXPIRE", req.Key.Key, req.Ttl)
			if err != nil {
				fastlog.Errorf("send expire failed, err:%v", err)
				return nil, err
			}
		}
	}

	return &resp, nil
}

func (s *RedisProxy) buildArgs(req *redisproxy.DoReq) ([]interface{}, bool, bool) {
	isMergedTtl, isMergedTtlSetnxCmd := s.isTtlMergeableCmd(req)
	argLen := len(req.Args) + 1
	if isMergedTtl {
		argLen += 2
	}
	args := make([]interface{}, 0, argLen)
	if req.Key.Key != "" {
		args = append(args, req.Key.Key)
	}
	for _, arg := range req.Args {
		switch arg.Type {
		case int32(redisproxy.Type_TypeString):
			args = append(args, arg.StrValue)
		case int32(redisproxy.Type_TypeInt64):
			args = append(args, arg.Int64Value)
		case int32(redisproxy.Type_TypeBytes):
			args = append(args, arg.BytesValue)
		case int32(redisproxy.Type_TypeUint64):
			args = append(args, arg.Uint64Value)
		case int32(redisproxy.Type_TypeFloat64):
			args = append(args, arg.FloatValue)
		}
	}
	if isMergedTtl {
		args = append(args, "EX", req.Ttl)
	}
	return args, isMergedTtl, isMergedTtlSetnxCmd
}

func (s *RedisProxy) isTtlMergeableCmd(req *redisproxy.DoReq) (bool, bool) {
	if req.Ttl <= 0 {
		return false, false
	}
	if req.Command == "SET" || req.Command == "set" {
		return true, false
	}
	if req.Command == "SETNX" || req.Command == "setnx" {
		req.Command = "SET"
		req.Args = append(req.Args, &redisproxy.Any{
			Type:     int32(redisproxy.Type_TypeString),
			StrValue: "NX",
		})
		return true, true
	}
	return false, false
}

func (s *RedisProxy) decodeReply(reply interface{}, resp *redisproxy.DoResp) {
	switch val := reply.(type) {
	case int64:
		resp.Int64Value = val
		resp.Type = int32(redisproxy.Type_TypeInt64)
		return
	case []byte:
		resp.BytesValue = val
		resp.Type = int32(redisproxy.Type_TypeBytes)
		return
	case string:
		resp.StrValue = val
		resp.Type = int32(redisproxy.Type_TypeString)
		return
	case []interface{}:
		resp.Type = int32(redisproxy.Type_TypeAnyArray)
		resp.AnyArrayValue = make([]*redisproxy.Any, 0, len(val))
		for _, v := range val {
			if converted, ok := s.convertAny(v); ok {
				resp.AnyArrayValue = append(resp.AnyArrayValue, converted)
			}
		}
		return
	}
}

func (s *RedisProxy) convertAny(v interface{}) (*redisproxy.Any, bool) {
	switch vv := v.(type) {
	case int64:
		return &redisproxy.Any{
			Type:       int32(redisproxy.Type_TypeInt64),
			Int64Value: vv,
		}, true
	case []byte:
		return &redisproxy.Any{
			Type:       int32(redisproxy.Type_TypeBytes),
			BytesValue: vv,
		}, true
	case string:
		return &redisproxy.Any{
			Type:     int32(redisproxy.Type_TypeString),
			StrValue: vv,
		}, true
	case []interface{}:
		var bytesArray [][]byte
		for _, vvv := range vv {
			switch vvv.(type) {
			case []byte:
				bytesArray = append(bytesArray, vvv.([]byte))
			}
		}
		return &redisproxy.Any{
			Type:            int32(redisproxy.Type_TypeBytesArray),
			BytesArrayValue: bytesArray,
		}, true
	}
	return nil, false
}
