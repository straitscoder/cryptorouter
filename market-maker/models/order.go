package model

import (
	"time"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

type Order struct {
	ClientOrderID string    `json:"clientOrderId" gorm:"primary_key"`
	OrderID       string    `json:"orderId" gorm:"unique"`
	ClientID      string    `json:"clientId"`
	Exchange      string    `json:"exchange"`
	Base          string    `json:"base"`
	Quote         string    `json:"quote"`
	Delimiter     string    `json:"delimiter"`
	Side          string    `json:"side"`
	AssetType     string    `json:"assetType"`
	OrderType     string    `json:"orderType"`
	Price         float64   `json:"price" gorm:"type:numeric(12,8)"`
	Amount        float64   `json:"amount" gorm:"type:numeric(12,8)"`
	Status        string    `json:"status"`
	Description   string    `json:"description"`
	Timestamp     time.Time `json:"timestamp"`
	Trades        []Trade   `json:"trades" gorm:"foreignKey:OrderID;references:OrderID"`
}

func GetOrders(cond *Order) (orders []Order) {
	db.Model(&Order{}).Where(cond).Preload("Trades").Find(&orders)
	return
}

func GetUnFilledOrders(cond *Order) (orders []Order) {
	db.Model(&Order{}).Not(map[string]interface{}{"status": []string{"FILLED", "CANCELLED"}}).Where(cond).Preload("Trades").Find(&orders)
	return
}

func CreateOrder(o Order) error {
	if err := db.Create(&o).Error; err != nil {
		return err
	}
	return nil
}

func GetOrderByClOrdID(clOrdID string) (order Order) {
	db.Model(&Order{}).Where(&Order{ClientOrderID: clOrdID}).Preload("Trades").First(&order)
	return
}

func GetOrderByOrderID(orderID string) (order Order) {
	db.Model(&Order{}).Where(&Order{OrderID: orderID}).Preload("Trades").First(&order)
	return
}

func UpdateOrder(clOrdId string, order Order) error {
	if err := db.Model(&Order{}).Where(&Order{ClientOrderID: clOrdId}).Updates(order).Error; err != nil {
		return err
	}
	return nil
}

func UpdateOrCreateOrder(orderDetail order.Detail, description string) error {
	order, trades := ToOrder(orderDetail, description)
	existingOrder := GetOrderByOrderID(order.OrderID)
	if existingOrder.ClientOrderID == "" {
		if len(trades) > 0 {
			for x := range trades {
				if err := UpdateOrCreateTrade(trades[x].TradeID, trades[x]); err != nil {
					return err
				}
			}
		}
		return CreateOrder(order)
	}

	if len(trades) > 0 {
		for x := range trades {
			if err := UpdateOrCreateTrade(trades[x].TradeID, trades[x]); err != nil {
				return err
			}
		}
	}

	return UpdateOrder(existingOrder.ClientOrderID, order)
}

func ToOrder(orderDetail order.Detail, description string) (Order, []Trade) {
	var trades []Trade
	if len(orderDetail.Trades) > 0 {
		trades = make([]Trade, len(orderDetail.Trades))
		for x := range orderDetail.Trades {
			trades[x] = Trade{
				TradeID:   orderDetail.Trades[x].TID,
				OrderID:   orderDetail.OrderID,
				Exchange:  orderDetail.Exchange,
				Price:     orderDetail.Trades[x].Price,
				Quantity:  orderDetail.Trades[x].Amount,
				Fee:       orderDetail.Trades[x].Fee,
				FeeAsset:  orderDetail.Trades[x].FeeAsset,
				Timestamp: orderDetail.Trades[x].Timestamp,
			}
		}
	}
	return Order{
		ClientOrderID: orderDetail.ClientOrderID,
		OrderID:       orderDetail.OrderID,
		ClientID:      orderDetail.ClientID,
		Exchange:      orderDetail.Exchange,
		Base:          orderDetail.Pair.Base.String(),
		Quote:         orderDetail.Pair.Quote.String(),
		Delimiter:     orderDetail.Pair.Delimiter,
		Side:          orderDetail.Side.String(),
		AssetType:     orderDetail.AssetType.String(),
		OrderType:     orderDetail.Type.String(),
		Price:         orderDetail.Price,
		Amount:        orderDetail.Amount,
		Status:        orderDetail.Status.String(),
		Description:   description,
		Timestamp:     orderDetail.Date,
	}, trades
}
