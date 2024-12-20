package model

import "time"

type Order struct {
	ClientOrderID string    `json:"clientOrderId" gorm:"primary_key"`
	OrderID       string    `json:"orderId" gorm:"unique"`
	SessionID     string    `json:"sessionId"`
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

func UpdateOrCreateOrder(order Order) error {
	existingOrder := GetOrderByOrderID(order.OrderID)
	if existingOrder.ClientOrderID == "" {
		return CreateOrder(order)
	}

	return UpdateOrder(existingOrder.ClientOrderID, order)
}
