package model

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
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

func AddExecutionReport(ctx context.Context, orderDetail *order.Detail, source string) error {
	rpcDetail := ToRpcOrderDetail(orderDetail)
	rpcDetail.Success = true
	// lousy way to know where repeated execution report come from
	// rpcDetail.Error = source
	binary, err := proto.Marshal(rpcDetail)
	if err != nil {
		return err
	}

	return rdClient.RPush(ctx, executionReportQueue, binary).Err()
}

func AddRejectExecutionReport(ctx context.Context, o *order.Detail, orderError error) error {
	if o == nil {
		return errors.New("order detail was nil")
	}
	rpcDetail := ToRpcOrderDetail(o)
	rpcDetail.Success = false
	rpcDetail.Error = orderError.Error()
	binary, err := proto.Marshal(rpcDetail)
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

func ToRpcOrderDetail(od *order.Detail) *gctrpc.OrderDetails {
	var trades []*gctrpc.TradeHistory
	if len(od.Trades) > 0 {
		trades = make([]*gctrpc.TradeHistory, len(od.Trades))
		for i := range od.Trades {
			trades[i] = &gctrpc.TradeHistory{
				Id:           od.Trades[i].TID,
				CreationTime: od.Trades[i].Timestamp.Unix(),
				Price:        od.Trades[i].Price,
				Amount:       od.Trades[i].Amount,
				Exchange:     od.Exchange,
				AssetType:    od.AssetType.String(),
				OrderSide:    od.Trades[i].Side.String(),
				Fee:          od.Trades[i].Fee,
				Total:        od.Trades[i].Total,
			}
		}
	}
	return &gctrpc.OrderDetails{
		Exchange:        od.Exchange,
		ClientOrderId:   od.ClientOrderID,
		Id:              od.OrderID,
		BaseCurrency:    od.Pair.Base.String(),
		QuoteCurrency:   od.Pair.Quote.String(),
		AssetType:       od.AssetType.String(),
		OrderSide:       od.Side.String(),
		OrderType:       od.Type.String(),
		CreationTime:    od.Date.Format(time.RFC3339),
		UpdateTime:      od.LastUpdated.Format(time.RFC3339),
		Status:          od.Status.String(),
		Price:           od.Price,
		AveragePrice:    od.AverageExecutedPrice,
		Amount:          od.Amount,
		FilledAmount:    od.ExecutedAmount,
		RemainingAmount: od.RemainingAmount,
		Fee:             od.Fee,
		Cost:            od.ExecutedAmount * od.AverageExecutedPrice,
		Trades:          trades,
	}
}
