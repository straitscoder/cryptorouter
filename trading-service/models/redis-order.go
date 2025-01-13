package model

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type OrderRedis struct {
	ClientOrderID   string  `redis:"clientOrderId"`
	OrderID         string  `redis:"orderId"`
	Exchange        string  `redis:"exchange"`
	Base            string  `redis:"base"`
	Quote           string  `redis:"quote"`
	Delimiter       string  `redis:"delimiter"`
	Side            string  `redis:"side"`
	AssetType       string  `redis:"assetType"`
	OrderType       string  `redis:"orderType"`
	Price           float64 `redis:"price"`
	AveragePrice    float64 `redis:"averagePrice"`
	Amount          float64 `redis:"amount"`
	FilledAmount    float64 `redis:"filledAmount"`
	RemainingAmount float64 `redis:"remainingAmount"`
	Status          string  `redis:"status"`
	Timestamp       int64   `redis:"timestamp"`
}

const (
	orderIDListKey = "orderIDList"
	orderKey       = "order"
)

func ToOrderRedis(detail order.Detail) OrderRedis {
	return OrderRedis{
		ClientOrderID:   detail.ClientOrderID,
		OrderID:         detail.OrderID,
		Exchange:        detail.Exchange,
		Base:            detail.Pair.Base.String(),
		Quote:           detail.Pair.Quote.String(),
		Delimiter:       detail.Pair.Delimiter,
		Side:            detail.Side.String(),
		AssetType:       detail.AssetType.String(),
		OrderType:       detail.Type.String(),
		Price:           detail.Price,
		AveragePrice:    detail.AverageExecutedPrice,
		Amount:          detail.Amount,
		FilledAmount:    detail.ExecutedAmount,
		RemainingAmount: detail.RemainingAmount,
		Status:          detail.Status.String(),
		Timestamp:       detail.Date.Unix(),
	}
}

func ToOrderDetail(orderR OrderRedis, trades []order.TradeHistory) (order.Detail, error) {
	pair := currency.NewPairWithDelimiter(orderR.Base, orderR.Quote, orderR.Delimiter)
	side, err := order.StringToOrderSide(orderR.Side)
	if err != nil {
		return order.Detail{}, err
	}
	oType, err := order.StringToOrderType(orderR.OrderType)
	if err != nil {
		return order.Detail{}, err
	}
	status, err := order.StringToOrderStatus(orderR.Status)
	if err != nil {
		return order.Detail{}, err
	}
	a, err := asset.New(orderR.AssetType)
	if err != nil {
		return order.Detail{}, err
	}

	return order.Detail{
		ClientOrderID:        orderR.ClientOrderID,
		OrderID:              orderR.OrderID,
		Exchange:             orderR.Exchange,
		Pair:                 pair,
		AssetType:            a,
		Type:                 oType,
		Side:                 side,
		Status:               status,
		Price:                orderR.Price,
		AverageExecutedPrice: orderR.AveragePrice,
		Amount:               orderR.Amount,
		ExecutedAmount:       orderR.FilledAmount,
		RemainingAmount:      orderR.RemainingAmount,
		Date:                 time.Unix(orderR.Timestamp, 0),
		Trades:               trades,
	}, nil
}

func AddOrderRedis(ctx context.Context, o order.Detail) error {
	orderData := ToOrderRedis(o)

	//Create Order ID List
	if err := rdClient.RPush(ctx, orderIDListKey, orderData.OrderID).Err(); err != nil {
		return err
	}
	// saved order as byte
	if err := rdClient.HSet(ctx, orderKey+":"+orderData.OrderID, orderData).Err(); err != nil {
		return err
	}

	if len(o.Trades) > 0 {
		err := UpdateOrCreateTradesRedis(ctx, o)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func GetOrdersRedis(ctx context.Context, cond, notCond *order.Filter) (orders []order.Detail, err error) {
	orderIDList, err := rdClient.LRange(ctx, orderIDListKey, 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return orders, nil
		}

		return orders, err
	}

	for x := range orderIDList {
		order, err := GetOrderRedis(ctx, orderIDList[x])
		if err != nil {
			return orders, err
		}

		if cond != nil {
			if !order.MatchFilter(cond) {
				continue
			}
		}

		if notCond != nil {
			if order.MatchFilter(notCond) {
				continue
			}
		}

		orders = append(orders, order)
	}

	return orders, nil
}

func GetOrderRedis(ctx context.Context, orderID string) (order order.Detail, err error) {
	var orderRedis OrderRedis
	err = rdClient.HGetAll(ctx, orderKey+":"+orderID).Scan(&orderRedis)
	if err != nil {
		if err == redis.Nil {
			return order, nil
		}

		return order, err
	}

	if orderRedis.ClientOrderID == "" {
		return order, nil
	}

	trades, err := GetTradesByOrderID(ctx, orderID)
	if err != nil {
		return order, err
	}
	return ToOrderDetail(orderRedis, trades)
}

func UpdateOrCreateOrderRedis(ctx context.Context, orderD order.Detail) error {
	existingOrder, err := GetOrderRedis(ctx, orderD.OrderID)
	if err != nil {
		return err
	}

	if existingOrder.ClientOrderID == "" {
		if err := AddOrderRedis(ctx, orderD); err != nil {
			return err
		}
		return nil
	}

	if len(orderD.Trades) != len(existingOrder.Trades) {
		if err := UpdateOrCreateTradesRedis(ctx, orderD); err != nil {
			return err
		}
	}

	existingOrder = orderD
	updatedOrder := ToOrderRedis(existingOrder)
	jsonUpdatedOrder, err := json.Marshal(updatedOrder)
	if err != nil {
		return err
	}

	if err := rdClient.Set(ctx, orderKey+":"+existingOrder.OrderID, jsonUpdatedOrder, 0).Err(); err != nil {
		return err
	}
	return nil
}
