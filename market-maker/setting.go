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
	priceLevels []float64

	quantityLevel1 float64
	quantityLevel2 float64
	quantityLevel3 float64
	quantityLevel4 float64
	quantityLevel5 float64
	quantityLevels []float64

	priceLevelDepth   int
	priceGapTolerance float64

	host     string
	port     int
	username string
	password string
	exchCCX  string
)

func init() {
	var err error
	Cfg, err = ini.Load("./price.cfg")
	if err != nil {
		log.Fatalf("error when loading config: %+v", err)
		return
	}
	LoadPrice()
	LoadThriftServer()
}

func LoadPrice() {
	sec, err := Cfg.GetSection("PRICECONF")
	if err != nil {
		log.Fatalf("error when loading price section: %+v", err)
		return
	}

	priceLevel1 = sec.Key("PriceLevel1").MustFloat64(2)
	priceLevels = append(priceLevels, priceLevel1)
	priceLevel2 = sec.Key("PriceLevel2").MustFloat64(4)
	priceLevels = append(priceLevels, priceLevel2)
	priceLevel3 = sec.Key("PriceLevel3").MustFloat64(6)
	priceLevels = append(priceLevels, priceLevel3)
	priceLevel4 = sec.Key("PriceLevel4").MustFloat64(8)
	priceLevels = append(priceLevels, priceLevel4)
	priceLevel5 = sec.Key("PriceLevel5").MustFloat64(10)
	priceLevels = append(priceLevels, priceLevel5)

	quantityLevel1 = sec.Key("QuantityLeve1").MustFloat64(1)
	quantityLevels = append(quantityLevels, quantityLevel1)
	quantityLevel2 = sec.Key("QuantityLeve2").MustFloat64(2)
	quantityLevels = append(quantityLevels, quantityLevel2)
	quantityLevel3 = sec.Key("QuantityLeve3").MustFloat64(3)
	quantityLevels = append(quantityLevels, quantityLevel3)
	quantityLevel4 = sec.Key("QuantityLeve4").MustFloat64(4)
	quantityLevels = append(quantityLevels, quantityLevel4)
	quantityLevel5 = sec.Key("QuantityLeve5").MustFloat64(5)
	quantityLevels = append(quantityLevels, quantityLevel5)

	priceLevelDepth = sec.Key("PriceLevelDepth").MustInt(5)
	priceGapTolerance = sec.Key("PriceGapTolerance").MustFloat64(0.05)
}

func LoadThriftServer() {
	sec, err := Cfg.GetSection("THRIFTSERVER")
	if err != nil {
		log.Fatalf("error load thrift server config: %+v", err)
		return
	}

	host = sec.Key("Host").MustString("192.168.1.50")
	port = sec.Key("Port").MustInt(9090)
	username = sec.Key("Username").MustString("@@!@!@!!2")
	password = sec.Key("Password").MustString("@!@!#!@@!@!")
	exchCCX = sec.Key("Exchange").MustString("CCX")
}
