package model

type Trade struct {
	TradeID        int64   `json:"tradeId"`
	OrderID        int64   `json:"orderId"`
	Price          float64 `json:"price" gorm:"type:numeric(12,8)"`
	Quantity       float64 `json:"qty" gorm:"type:numeric(12,8)"`
	Commision      float64 `json:"commision" gorm:"type:numeric(12,8)"`
	CommisionAsset string  `json:"commisionAsset"`
}

func AddTrade(trade Trade) error {
	return db.Create(&trade).Error
}

func GetTrades(cond *Trade) (trades []Trade) {
	db.Model(&Trade{}).Where(cond).Find(&trades)
	return
}
