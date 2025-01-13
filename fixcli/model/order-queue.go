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

func GetExecutionReportQueue(ctx context.Context) (*gctrpc.OrderDetails, error) {
	binary, err := rdClient.LPop(ctx, executionReportQueue).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var orderDetail gctrpc.OrderDetails
	return &orderDetail, proto.Unmarshal(binary, &orderDetail)
}
