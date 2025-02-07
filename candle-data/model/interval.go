package model

import "time"

type Interval struct {
	ID           int       `json:"id" gorm:"primaryKey,autoIncrement"`
	InstrumentID int       `json:"instrument_id"`
	Interval     string    `json:"interval"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime"`
	Candles      []Candle  `json:"candles" gorm:"foreignKey:IntervalID"`
}

func CreateInterval(in Interval) error {
	if len(in.Candles) > 0 {
		for i := range in.Candles {
			if err := CreateCandle(in.Candles[i]); err != nil {
				return err
			}
		}
	}
	clear(in.Candles)
	return db.Create(&in).Error
}

func GetIntervals(cond *Interval) (intervals []Interval) {
	db.Model(Interval{}).Where(cond).Preload("Candles").Find(&intervals)
	return
}

func GetInterval(id int) (interval Interval) {
	db.Model(Interval{}).Where("id = ?", id).Preload("Candles").First(&interval)
	return
}
