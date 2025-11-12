package redisproxy

import (
	"context"

	easymicrogrpc "github.com/995933447/easymicro/grpc"
	"github.com/gomodule/redigo/redis"
	"google.golang.org/grpc"
)

func PrepareGRPC(discoveryName string, opts ...grpc.DialOption) error {
	if err := easymicrogrpc.PrepareDiscoverGRPC(context.TODO(), EasymicroGRPCSchema, discoveryName); err != nil {
		return err
	}
	easymicrogrpc.RegisterServiceDialOpts(EasymicroGRPCPbServiceNameRedisProxy, true, opts...)
	return nil
}

func DoCmd(ctx context.Context, isAsync bool, ttl int64, connName, command, key string, args ...interface{}) (reply interface{}, e error) {
	req := &DoReq{
		Command: command,
		IsAsync: isAsync,
		Key: &RedisKey{
			Conn: connName,
			Key:  key,
		},
		Ttl: ttl,
	}
	for _, arg := range args {
		switch arg.(type) {
		case []byte:
			req.Args = append(req.Args, &Any{
				BytesValue: arg.([]byte),
				Type:       int32(Type_TypeBytes),
			})
		case string:
			req.Args = append(req.Args, &Any{
				StrValue: arg.(string),
				Type:     int32(Type_TypeString),
			})
		case int:
			req.Args = append(req.Args, &Any{
				Int64Value: int64(arg.(int)),
				Type:       int32(Type_TypeInt64),
			})
		case int8:
			req.Args = append(req.Args, &Any{
				Int64Value: int64(arg.(int8)),
				Type:       int32(Type_TypeInt64),
			})
		case int16:
			req.Args = append(req.Args, &Any{
				Int64Value: int64(arg.(int16)),
				Type:       int32(Type_TypeInt64),
			})
		case int32:
			req.Args = append(req.Args, &Any{
				Int64Value: int64(arg.(int32)),
				Type:       int32(Type_TypeInt64),
			})
		case int64:
			req.Args = append(req.Args, &Any{
				Int64Value: arg.(int64),
				Type:       int32(Type_TypeInt64),
			})
		case uint:
			req.Args = append(req.Args, &Any{
				Uint64Value: uint64(arg.(uint)),
				Type:        int32(Type_TypeUint64),
			})
		case uint8:
			req.Args = append(req.Args, &Any{
				Uint64Value: uint64(arg.(uint8)),
				Type:        int32(Type_TypeUint64),
			})
		case uint16:
			req.Args = append(req.Args, &Any{
				Int64Value: int64(arg.(int16)),
				Type:       int32(Type_TypeUint64),
			})
		case uint32:
			req.Args = append(req.Args, &Any{
				Uint64Value: uint64(arg.(uint32)),
				Type:        int32(Type_TypeUint64),
			})
		case uint64:
			req.Args = append(req.Args, &Any{
				Uint64Value: arg.(uint64),
				Type:        int32(Type_TypeUint64),
			})
		case float32:
			req.Args = append(req.Args, &Any{
				FloatValue: float64(arg.(float32)),
				Type:       int32(Type_TypeFloat64),
			})
		case float64:
			req.Args = append(req.Args, &Any{
				FloatValue: arg.(float64),
				Type:       int32(Type_TypeFloat64),
			})
		}
	}

	doResp, err := RedisProxyGRPC().Do(ctx, req)
	if err != nil {
		if easymicrogrpc.IsRPCErr(err, ErrCode_ErrCodeRedisNil.Number()) {
			return nil, redis.ErrNil
		}

		return nil, err
	}

	switch doResp.Type {
	case int32(Type_TypeString):
		return doResp.StrValue, nil
	case int32(Type_TypeInt64):
		return doResp.Int64Value, nil
	case int32(Type_TypeBytes):
		return doResp.BytesValue, nil
	case int32(Type_TypeAnyArray):
		var res []interface{}
		for _, v := range doResp.AnyArrayValue {
			switch v.Type {
			case int32(Type_TypeString):
				res = append(res, v.StrValue)
			case int32(Type_TypeInt64):
				res = append(res, v.Int64Value)
			case int32(Type_TypeBytes):
				res = append(res, v.BytesValue)
			case int32(Type_TypeBytesArray):
				var bytes2InterfaceArr []interface{}
				for _, vv := range v.BytesArrayValue {
					bytes2InterfaceArr = append(bytes2InterfaceArr, vv)
				}
				res = append(res, bytes2InterfaceArr)
			}
		}
		return res, nil
	}

	return nil, nil
}
