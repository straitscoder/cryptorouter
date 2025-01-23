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
	ClientOrderID   string  `json:"clientOrderId"`
	OrderID         string  `json:"orderId"`
	Exchange        string  `json:"exchange"`
	Base            string  `json:"base"`
	Quote           string  `json:"quote"`
	Delimiter       string  `json:"delimiter"`
	Side            string  `json:"side"`
	AssetType       string  `json:"assetType"`
	OrderType       string  `json:"orderType"`
	Price           float64 `json:"price"`
	AveragePrice    float64 `json:"averagePrice"`
	Amount          float64 `json:"amount"`
	FilledAmount    float64 `json:"filledAmount"`
	RemainingAmount float64 `json:"remainingAmount"`
	Status          string  `json:"status"`
	Timestamp       int64   `json:"timestamp"`
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
	jsonOrder, err := json.Marshal(orderData)
	if err != nil {
		return err
	}

	orderHash, err := rdClient.HGetAll(ctx, orderKey).Result()
	if err != nil {
		if err == redis.Nil {
			orderHash = make(map[string]string)
			orderHash[orderData.OrderID] = string(jsonOrder)
			//Create Order ID List
			if err := rdClient.RPush(ctx, orderIDListKey, orderData.OrderID).Err(); err != nil {
				return err
			}
			// saved order as byte
			if err := rdClient.HSet(ctx, orderKey, orderHash).Err(); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	//Create Order ID List
	if err := rdClient.RPush(ctx, orderIDListKey, orderData.OrderID).Err(); err != nil {
		return err
	}
	// saved order as byte
	if err := rdClient.HSetNX(ctx, orderKey, o.OrderID, jsonOrder).Err(); err != nil {
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
	jsonOrder, err := rdClient.HGet(ctx, orderKey, orderID).Result()
	if err != nil {
		if err == redis.Nil {
			return order, nil
		}

		return order, err
	}

	if jsonOrder == "" {
		return order, nil
	}

	var orderRedis OrderRedis
	if err := json.Unmarshal([]byte(jsonOrder), &orderRedis); err != nil {
		return order, err
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

	updatedOrder := ToOrderRedis(orderD)
	jsonUpdatedOrder, err := json.Marshal(updatedOrder)
	if err != nil {
		return err
	}

	orderHash, err := rdClient.HGetAll(ctx, orderKey).Result()
	if err != nil {
		return err
	}

	orderHash[updatedOrder.OrderID] = string(jsonUpdatedOrder)
	if err := rdClient.HSet(ctx, orderKey, orderHash).Err(); err != nil {
		return err
	}

	return nil
}
