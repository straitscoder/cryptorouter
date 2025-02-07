package model

import "time"

type Instrument struct {
	ID        int        `json:"id" gorm:"primaryKey,autoIncrement"`
	Exchange  string     `json:"exchange"`
	AssetType string     `json:"asset_type"`
	Symbol    string     `json:"symbol"`
	CreatedAt time.Time  `json:"created_at" gorm:"autoCreateTime"`
	Intervals []Interval `json:"intervals" gorm:"foreignKey:IntervalID"`
}

func CreateInstrument(in Instrument) error {
	if len(in.Intervals) > 0 {
		for i := range in.Intervals {
			if err := CreateInterval(in.Intervals[i]); err != nil {
				return err
			}
		}
	}

	clear(in.Intervals)
	return db.Create(&in).Error
}

func GetInstruments(cond *Instrument) (instruments []Instrument) {
	db.Model(&Instrument{}).Where(cond).Preload("Intervals.Candles").Find(&instruments)
	return
}

func GetInstrument(id int) (instrument Instrument) {
	db.Model(&Instrument{}).Where("id = ?", id).Preload("Intervals.Candles").First(&instrument)
	return
}

func GetInstrumentWithCond(cond *Instrument) (instrument Instrument) {
	db.Model(&Instrument{}).Where(cond).Preload("Intervals.Candles").First(&instrument)
	return
}

func UpdateOrCreateInstrument(in Instrument) error {
	existingInstrument := GetInstrumentWithCond(&Instrument{Exchange: in.Exchange, AssetType: in.AssetType, Symbol: in.Symbol})
	if existingInstrument.ID == 0 {
		return CreateInstrument(in)
	}

	in.ID = existingInstrument.ID
	return db.Model(&Instrument{}).Where("id = ?", in.ID).Updates(in).Error
}
