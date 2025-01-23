package model

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type TradeRedis struct {
	TradeID         string  `json:"tradeId" gorm:"primary_key"`
	OrderID         string  `json:"orderId" gorm:"primary_key"`
	Exchange        string  `json:"exchange"`
	Side            string  `json:"side"`
	Price           float64 `json:"price" gorm:"type:numeric(12,8)"`
	Quantity        float64 `json:"qty" gorm:"type:numeric(12,8)"`
	Commission      float64 `json:"commission" gorm:"type:numeric(12,8)"`
	CommissionAsset string  `json:"commissionAsset"`
	Timestamp       int64   `json:"timestamp"`
	Total           float64 `json:"total"`
}

const (
	tradeIDListKey       = "tradeIDList"
	tradeKey             = "trade"
	orderTradesIDListKey = "orderTrades"
)

func GenerateOrderTradesKey(orderId string) string {
	return fmt.Sprintf("%s:%s:%s", orderKey, orderId, tradeKey)
}

func GenerateTradeKey(tradeID, orderID string) string {
	return fmt.Sprintf("%s:%s:%s", tradeKey, tradeID, orderID)
}

func ToTradeRedis(tradeH order.TradeHistory, orderId string) TradeRedis {
	return TradeRedis{
		TradeID:         tradeH.TID,
		OrderID:         orderId,
		Exchange:        tradeH.Exchange,
		Side:            tradeH.Side.String(),
		Price:           tradeH.Price,
		Quantity:        tradeH.Amount,
		Commission:      tradeH.Fee,
		CommissionAsset: tradeH.FeeAsset,
		Timestamp:       tradeH.Timestamp.Unix(),
		Total:           tradeH.Total,
	}
}

func ToTradesRedis(orderD order.Detail) []TradeRedis {
	trades := make([]TradeRedis, len(orderD.Trades))
	for i := range orderD.Trades {
		trades[i] = TradeRedis{
			TradeID:         orderD.Trades[i].TID,
			OrderID:         orderD.OrderID,
			Exchange:        orderD.Exchange,
			Side:            orderD.Trades[i].Side.String(),
			Price:           orderD.Trades[i].Price,
			Quantity:        orderD.Trades[i].Amount,
			Commission:      orderD.Trades[i].Fee,
			CommissionAsset: orderD.Trades[i].FeeAsset,
			Timestamp:       orderD.Trades[i].Timestamp.Unix(),
			Total:           orderD.Trades[i].Total,
		}
	}
	return trades
}

func ToTradeHistory(trade TradeRedis) (order.TradeHistory, error) {
	side, err := order.StringToOrderSide(trade.Side)
	if err != nil {
		return order.TradeHistory{}, err
	}
	return order.TradeHistory{
		TID:       trade.TradeID,
		Exchange:  trade.Exchange,
		Side:      side,
		Price:     trade.Price,
		Amount:    trade.Quantity,
		Fee:       trade.Commission,
		FeeAsset:  trade.CommissionAsset,
		Timestamp: time.Unix(trade.Timestamp, 0),
		Total:     trade.Total,
	}, nil
}

func AddTradeRedis(ctx context.Context, trade TradeRedis) error {
	jsonTrade, err := json.Marshal(trade)
	if err != nil {
		return err
	}

	tradeHash, err := rdClient.HGetAll(ctx, tradeKey).Result()
	if err != nil {
		if err == redis.Nil {
			tradeHash = make(map[string]string)
			if err := rdClient.RPush(ctx, tradeIDListKey, trade.TradeID).Err(); err != nil {
				return err
			}

			if err := rdClient.RPush(ctx, GenerateOrderTradesKey(trade.OrderID), trade.TradeID).Err(); err != nil {
				return err
			}

			tradeHash[fmt.Sprintf("%s:%s", trade.TradeID, trade.OrderID)] = string(jsonTrade)
			if err := rdClient.HSet(ctx, tradeKey, tradeHash).Err(); err != nil {
				return err
			}
			return nil
		}
		return err
	}

	if err := rdClient.RPush(ctx, tradeIDListKey, trade.TradeID).Err(); err != nil {
		return err
	}

	if err := rdClient.RPush(ctx, GenerateOrderTradesKey(trade.OrderID), trade.TradeID).Err(); err != nil {
		return err
	}

	if err := rdClient.HSetNX(ctx, tradeKey, fmt.Sprintf("%s:%s", trade.TradeID, trade.OrderID), jsonTrade).Err(); err != nil {
		return err
	}

	return nil
}

func GetTradeRedis(ctx context.Context, tradeId, orderId string) (order.TradeHistory, error) {
	jsonTrade, err := rdClient.HGet(ctx, tradeKey, fmt.Sprintf("%s:%s", tradeId, orderId)).Result()
	if err != nil {
		if err == redis.Nil {
			return order.TradeHistory{}, nil
		}
		return order.TradeHistory{}, err
	}

	var tradeRedis TradeRedis
	err = json.Unmarshal([]byte(jsonTrade), &tradeRedis)
	if err != nil {
		return order.TradeHistory{}, err
	}

	if tradeRedis.Exchange == "" {
		return order.TradeHistory{}, nil
	}

	return ToTradeHistory(tradeRedis)
}

func UpdateOrCreateTradeRedis(ctx context.Context, trade TradeRedis) error {
	existingTrade, err := GetTradeRedis(ctx, trade.TradeID, trade.OrderID)
	if err != nil {
		return err
	}

	if existingTrade.Exchange == "" {
		if err := AddTradeRedis(ctx, trade); err != nil {
			return err
		}
		return nil
	}
	// Update existing data
	jsonTrade, err := json.Marshal(trade)
	if err != nil {
		return err
	}

	tradeHash, err := rdClient.HGetAll(ctx, tradeKey).Result()
	if err != nil {
		return err
	}

	tradeHash[fmt.Sprintf("%s:%s", trade.TradeID, trade.OrderID)] = string(jsonTrade)
	if err := rdClient.HSet(ctx, tradeKey, tradeHash).Err(); err != nil {
		return err
	}

	return nil
}

func GetTradesByOrderID(ctx context.Context, orderId string) ([]order.TradeHistory, error) {
	var trades []order.TradeHistory
	tradeIDListByOrder, err := rdClient.LRange(ctx, GenerateOrderTradesKey(orderId), 0, -1).Result()
	if err != nil {
		if err == redis.Nil {
			return trades, nil
		}
		return trades, err
	}

	for i := range tradeIDListByOrder {
		trade, err := GetTradeRedis(ctx, tradeIDListByOrder[i], orderId)
		if err != nil {
			log.Printf("Error when get this trade %s: %+v", tradeIDListByOrder[i], err)
			continue
		}

		if trade.Exchange == "" {
			continue
		}

		trades = append(trades, trade)
	}
	return trades, nil
}

func UpdateOrCreateTradesRedis(ctx context.Context, orderDetail order.Detail) error {
	if len(orderDetail.Trades) < 1 {
		return nil
	}

	redisTrades := ToTradesRedis(orderDetail)

	for i := range redisTrades {
		if err := UpdateOrCreateTradeRedis(ctx, redisTrades[i]); err != nil {
			log.Printf("error when update or create this trade %s: %+v", redisTrades[i].TradeID, err)
			continue
		}
	}

	return nil
}
