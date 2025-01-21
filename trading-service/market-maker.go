package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	model "github.com/thrasher-corp/gocryptotrader/trading-service/models"
)

type PriceReference struct {
	Exchange  string
	AssetType string
	isFuture  bool
	Symbol    string
	Price     float64
	Volume    float64
}

type PriceReferences map[string]PriceReference

type MarketMaker struct {
	ExchangeManager *ExchangeManager
	PairFormatter   *currency.PairFormat
	Shutdown        chan struct{}
}

func NewMarketMaker(exchManager *ExchangeManager) (*MarketMaker, error) {
	if exchManager == nil {
		return nil, errors.New("exchange manager is nil")
	}
	marketMaker := MarketMaker{
		PairFormatter: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
	}
	marketMaker.Shutdown = make(chan struct{})
	marketMaker.ExchangeManager = exchManager
	return &marketMaker, nil
}

func (m *MarketMaker) Start() {
	m.run()
}

func (m *MarketMaker) Stop() {
	m.Shutdown <- struct{}{}
	close(m.Shutdown)
	return
}

func (m *MarketMaker) run() {
	priceReferences, err := m.GetFairPrice()
	if err != nil {
		log.Printf("get fair price error: %+v", err)
	}
	log.Printf("fair prices: %+v", priceReferences)
	ticker := time.NewTicker(time.Second * 2)
	defer ticker.Stop()

	for {
		select {
		case <-m.Shutdown:
			ticker.Stop()
			return
		case <-ticker.C:
			go func() {
				priceReferences, err = m.GetFairPrice()
				if err != nil {
					log.Printf("get fair price error: %+v", err)
				}
				log.Printf("fair prices: %+v", priceReferences)
			}()
		}
	}
}

func (m *MarketMaker) GetFairPrice() (PriceReferences, error) {
	result := make(PriceReferences)
	exchanges, err := m.ExchangeManager.GetExchanges()
	if err != nil {
		return result, err
	}

	if len(exchanges) == 0 {
		return result, nil
	}

	for x := range exchanges {
		if !exchanges[x].GetBase().Enabled {
			continue
		}
		enabledAssets := exchanges[x].GetAssetTypes(true)
		if len(enabledAssets) == 0 {
			continue
		}

		for y := range enabledAssets {
			pairs, err := exchanges[x].GetEnabledPairs(enabledAssets[y])
			if err != nil {
				log.Printf("Error when Get enbaled pair from %s %s: %+v", exchanges[x].GetName(), enabledAssets[y].String(), err)
				continue
			}

			if len(pairs) == 0 {
				continue
			}

			for z := range pairs {
				orderbook, err := exchanges[x].FetchOrderbook(context.TODO(), pairs[z], enabledAssets[y])
				if err != nil {
					log.Printf("Error when fetch %s order book from %s %s: %+v", pairs[z].String(), exchanges[x].GetName(), enabledAssets[y].String(), err)
					continue
				}

				symbol := m.PairFormatter.Format(orderbook.Pair)
				var fieldName string
				switch orderbook.Asset.IsFutures() {
				case true:
					fieldName = symbol + "-FUT"
				case false:
					fieldName = symbol + "-" + orderbook.Asset.String()
				}
				var bestBuyPrice float64
				var bestSellPrice float64
				var totalAskVolume decimal.Decimal
				var totalBidVolume decimal.Decimal
				for a := range orderbook.Asks {
					totalAskVolume = totalAskVolume.Add(decimal.NewFromFloatWithExponent(orderbook.Asks[a].Amount, -5))
					if bestSellPrice == 0 {
						bestSellPrice = orderbook.Asks[a].Price
					} else if bestSellPrice > orderbook.Asks[a].Price {
						bestSellPrice = orderbook.Asks[a].Price
					} else {
						continue
					}
				}

				for b := range orderbook.Bids {
					totalBidVolume = totalBidVolume.Add(decimal.NewFromFloatWithExponent(orderbook.Bids[b].Amount, -5))
					if bestBuyPrice == 0 {
						bestBuyPrice = orderbook.Bids[b].Price
					} else if bestBuyPrice > orderbook.Bids[b].Price {
						bestBuyPrice = orderbook.Bids[b].Price
					} else {
						continue
					}
				}

				fairPrice := (bestBuyPrice + bestSellPrice) / 2
				exchangeTotalVolume := decimal.Sum(totalAskVolume, totalBidVolume)
				if result[fieldName].Price == 0 && result[fieldName].Volume == 0 {
					result[fieldName] = PriceReference{
						Exchange:  orderbook.Exchange,
						AssetType: orderbook.Asset.String(),
						isFuture:  orderbook.Asset.IsFutures(),
						Symbol:    orderbook.Pair.String(),
						Price:     fairPrice,
						Volume:    exchangeTotalVolume.InexactFloat64(),
					}
				} else if result[fieldName].Volume < exchangeTotalVolume.InexactFloat64() {
					result[fieldName] = PriceReference{
						Exchange:  orderbook.Exchange,
						AssetType: orderbook.Asset.String(),
						isFuture:  orderbook.Asset.IsFutures(),
						Symbol:    orderbook.Pair.String(),
						Price:     fairPrice,
						Volume:    exchangeTotalVolume.InexactFloat64(),
					}
				} else {
					continue
				}
			}
		}
	}
	for key := range result {
		if result[key].Volume == 0 {
			delete(result, key)
		}
	}
	return result, nil
}

func generateRandomString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		log.Printf("error when create clien order id: %+v", err)
		return ""
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

func (m *MarketMaker) GenerateClOrdID() string {
	timestamp := time.Now().Unix()         // Unix timestamp for uniqueness
	randomPart := generateRandomString(10) // Random alphanumeric string
	clOrdId := fmt.Sprintf("%d%s", timestamp, randomPart)
	if len(clOrdId) > 36 {
		clOrdId = clOrdId[:36]
	}
	return clOrdId
}

func (m *MarketMaker) PlaceOrder() {
	fairPrices, err := m.GetFairPrice()
	if err != nil {
		log.Printf("error when get fair prices: %+v", err)
		return
	}

	for _, value := range fairPrices {
		pair, err := currency.NewPairFromString(value.Symbol)
		if err != nil {
			log.Printf("error converting fair price symbol: %+v", err)
			continue
		}

		createdOrders, err := model.GetOrdersRedis(context.Background(),
			&order.Filter{Exchange: value.Exchange, Pair: pair},
			&order.Filter{},
		)

		if err != nil {
			log.Printf("error getting created order: %+v", err)
			continue
		}

		if len(createdOrders) == 0 {

		}
	}
}

func GeneratePriceLevels(price float64, side string) []float64 {
	priceDepth := make([]float64, 3)
	side = strings.ToUpper(side)
	decimals := countDecimals(price)
	switch decimals {
	case 0:
		for i := range priceDepth {
			if side == "BID" {
				price -= 10
			} else {
				price += 10
			}
			priceDepth[i] = price
		}
	case 1:
		for i := range priceDepth {
			if side == "BID" {
				price -= 0.1
			} else {
				price += 0.1
			}
			priceDepth[i] = price
		}
	case 2:
		for i := range priceDepth {
			if side == "BID" {
				price -= 0.01
			} else {
				price += 0.01
			}
			priceDepth[i] = price
		}
	case 3:
		for i := range priceDepth {
			if side == "BID" {
				price -= 0.001
			} else {
				price += 0.001
			}
			priceDepth[i] = price
		}
	case 4:
		for i := range priceDepth {
			if side == "BID" {
				price -= 0.0001
			} else {
				price += 0.0001
			}
			priceDepth[i] = price
		}
	case 5:
		for i := range priceDepth {
			if side == "BID" {
				price -= 0.00001
			} else {
				price += 0.00001
			}
			priceDepth[i] = price
		}
	default:
		for i := range priceDepth {
			if side == "BID" {
				price -= 0.000001
			} else {
				price += 0.000001
			}
			priceDepth[i] = price
		}
	}
	return priceDepth
}

func countDecimals(number float64) int {
	nmbrStr := strconv.FormatFloat(number, 'f', -1, 64)

	parts := strings.Split(nmbrStr, ".")

	if len(parts) == 1 {
		return 0
	}

	return len(parts[1])
}

func GetNotionalAmout(price float64) float64 {
	notionalValue := float64(5)
	return notionalValue / price
}
