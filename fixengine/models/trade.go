package model

type Trade struct {
	TradeID        int64   `json:"tradeId" gorm:"primary_key"`
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

func GetTradesByTradeID(tradeID int64) (trades Trade) {
	db.Model(&Trade{}).Where(&Trade{TradeID: tradeID}).Find(&trades)
	return
}

func UpdateOrCreateTrade(tradeId int64, trade Trade) error {
	existingTrade := GetTradesByTradeID(tradeId)

	if existingTrade.OrderID == 0 {
		return AddTrade(trade)
	}

	if err := db.Model(&Trade{}).Where(&Trade{TradeID: tradeId}).Updates(trade).Error; err != nil {
		return err
	}

	return nil
}
