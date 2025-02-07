package model

import "time"

type Candle struct {
	ID         int       `json:"id" gorm:"primaryKey,autoIncrement"`
	IntervalID int       `json:"interval_id"`
	Open       float64   `json:"open" gorm:"type:numeric(12,8)"`
	High       float64   `json:"high" gorm:"type:numeric(12,8)"`
	Low        float64   `json:"low" gorm:"type:numeric(12,8)"`
	Close      float64   `json:"close" gorm:"type:numeric(12,8)"`
	Volume     float64   `json:"volume" gorm:"type:numeric(12,8)"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoCreateTime"`
}

func CreateCandle(candle Candle) error {
	return db.Create(&candle).Error
}

func GetCandles(cond *Candle) (candles []Candle) {
	db.Model(&Candle{}).Where(cond).Find(&candles)
	return
}

func GetCandle(id int) (candle Candle) {
	db.Model(&Candle{}).Where("id = ?", id).First(&candle)
	return
}
