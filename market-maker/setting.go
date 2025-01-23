package main

import (
	"log"

	"github.com/go-ini/ini"
)

var (
	Cfg *ini.File

	priceLevel1 float64
	priceLevel2 float64
	priceLevel3 float64
	priceLevel4 float64
	priceLevel5 float64

	decimalMultiplier  float64
	decimalMultiplier1 float64
	decimalMultiplier2 float64

	quantityLevel1 float64
	quantityLevel2 float64
	quantityLevel3 float64
	quantityLevel4 float64
	quantityLevel5 float64

	priceLevelDepth int
)

func init() {
	var err error
	Cfg, err = ini.Load("./price.cfg")
	if err != nil {
		log.Fatalf("error when loading config: %+v", err)
		return
	}
	LoadPrice()
}

func LoadPrice() {
	sec, err := Cfg.GetSection("PRICECONF")
	if err != nil {
		log.Fatalf("error when loading price section: %+v", err)
		return
	}

	priceLevel1 = sec.Key("PriceLevel1").MustFloat64(2)
	priceLevel2 = sec.Key("PriceLevel2").MustFloat64(4)
	priceLevel3 = sec.Key("PriceLevel3").MustFloat64(6)
	priceLevel4 = sec.Key("PriceLevel4").MustFloat64(8)
	priceLevel5 = sec.Key("PriceLevel5").MustFloat64(10)

	decimalMultiplier = sec.Key("DecimalMultiplier").MustFloat64(1)
	decimalMultiplier1 = sec.Key("DecimalMultiplier1").MustFloat64(0.1)
	decimalMultiplier2 = sec.Key("DecimalMultiplier2").MustFloat64(0.01)

	quantityLevel1 = sec.Key("QuantityLeve1").MustFloat64(1)
	quantityLevel2 = sec.Key("QuantityLeve2").MustFloat64(2)
	quantityLevel3 = sec.Key("QuantityLeve3").MustFloat64(3)
	quantityLevel4 = sec.Key("QuantityLeve4").MustFloat64(4)
	quantityLevel5 = sec.Key("QuantityLeve5").MustFloat64(5)

	priceLevelDepth = sec.Key("PriceLevelDepth").MustInt(5)
}
