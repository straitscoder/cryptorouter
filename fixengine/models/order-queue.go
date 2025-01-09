package model

import (
	"context"

	"github.com/redis/go-redis/v9"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/protobuf/proto"
)

const (
	submitOrderQueue     = "submitOrder"
	modifyOrderQueue     = "modifyOrder"
	cancelOrderQueue     = "cancelOrder"
	executionReportQueue = "executionReport"
)

func GetSubmitQueue(ctx context.Context) (*gctrpc.SubmitOrderRequest, error) {
	var request gctrpc.SubmitOrderRequest
	binary, err := rdClient.LPop(ctx, submitOrderQueue).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	return &request, proto.Unmarshal(binary, &request)
}

func GetModifyQueue(ctx context.Context) (*gctrpc.ModifyOrderRequest, error) {
	var request gctrpc.ModifyOrderRequest
	binary, err := rdClient.LPop(ctx, modifyOrderQueue).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	return &request, proto.Unmarshal(binary, &request)
}

func GetCancelQueue(ctx context.Context) (*gctrpc.CancelOrderRequest, error) {
	var request gctrpc.CancelOrderRequest
	binary, err := rdClient.LPop(ctx, cancelOrderQueue).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	return &request, proto.Unmarshal(binary, &request)
}

func AddExecutionReport(ctx context.Context, orderDetail *gctrpc.OrderDetails) error {
	binary, err := proto.Marshal(orderDetail)
	if err != nil {
		return err
	}

	return rdClient.RPush(ctx, executionReportQueue, binary).Err()
}

func AddSubmitQueue(ctx context.Context, req *gctrpc.SubmitOrderRequest) error {
	binary, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	if err := rdClient.RPush(ctx, submitOrderQueue, binary).Err(); err != nil {
		return err
	}
	return nil
}

func AddModifyQueue(ctx context.Context, req *gctrpc.ModifyOrderRequest) error {
	binary, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	if err := rdClient.RPush(ctx, modifyOrderQueue, binary).Err(); err != nil {
		return err
	}

	return nil
}

func AddCancelQueue(ctx context.Context, req *gctrpc.CancelOrderRequest) error {
	binary, err := proto.Marshal(req)
	if err != nil {
		return err
	}

	if err := rdClient.RPush(ctx, cancelOrderQueue, binary).Err(); err != nil {
		return err
	}
	return nil
}
