package main

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/trading-service/fixengine"
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
	ProcessingOrder int32
	FixEngine       *fixengine.FixEngine
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
	marketMaker.FixEngine = new(fixengine.FixEngine)
	return &marketMaker, nil
}

func (m *MarketMaker) Start() error {
	if err := m.FixEngine.Start(); err != nil {
		return err
	}
	if err := m.FixEngine.SecuritiesDetail(); err != nil {
		log.Printf("error when requesting security detail: %+v", err)
		return err
	}
	go m.run()
	return nil
}

func (m *MarketMaker) Stop() {
	m.ShutdownRoutine()
	m.FixEngine.Stop()
	m.Shutdown <- struct{}{}
	close(m.Shutdown)
	return
}

func (m *MarketMaker) run() {
	m.PlaceOrder()
	ticker := time.NewTicker(time.Second * 1)
	defer ticker.Stop()

	for {
		select {
		case <-m.Shutdown:
			ticker.Stop()
			return
		case <-ticker.C:
			go m.PlaceOrder()
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
			pairs, err := exchanges[x].GetAvailablePairs(enabledAssets[y])
			if err != nil {
				log.Printf("Error when Get enbaled pair: %+v", err)
				continue
			}

			ccxPairs, err := m.FixEngine.GetCCXPairs()
			if err != nil {
				log.Print(err)
				continue
			}

			if len(ccxPairs) == 0 {
				if err := m.FixEngine.SecuritiesDetail(); err != nil {
					log.Print(err)
					continue
				}
				continue
			}

			if len(pairs) == 0 {
				continue
			}

			for z := range ccxPairs {
				if enabledAssets[y] != asset.Spot {
					continue
				}

				orderbook, err := exchanges[x].FetchOrderbook(context.TODO(), ccxPairs[z], enabledAssets[y])
				if err != nil {
					// log.Printf("Error when fetch %s order book from %s %s: %+v", pairs[z].String(), exchanges[x].GetName(), enabledAssets[y].String(), err)
					continue
				}

				symbol := m.PairFormatter.Format(orderbook.Pair)
				quoteCurrency := strings.Split(orderbook.Pair.Quote.String(), "-")[0]
				var fieldName string
				switch orderbook.Asset.IsFutures() {
				case true:
					fieldName = orderbook.Pair.Base.String() + "-" + quoteCurrency
				case false:
					fieldName = symbol
				}
				var bestBuyPrice float64
				var bestSellPrice float64
				var totalAskVolume decimal.Decimal
				var totalBidVolume decimal.Decimal
				var totalVolumeFloat float64
				for a := range orderbook.Asks {
					totalAskVolume = totalAskVolume.Add(decimal.NewFromFloatWithExponent(orderbook.Asks[a].Amount, -8))
					totalVolumeFloat += orderbook.Asks[a].Amount
					if bestSellPrice == 0 {
						bestSellPrice = orderbook.Asks[a].Price
					} else if bestSellPrice > orderbook.Asks[a].Price {
						bestSellPrice = orderbook.Asks[a].Price
					} else {
						continue
					}
				}

				for b := range orderbook.Bids {
					totalBidVolume = totalBidVolume.Add(decimal.NewFromFloatWithExponent(orderbook.Bids[b].Amount, -8))
					totalVolumeFloat += orderbook.Bids[b].Amount
					if bestBuyPrice == 0 {
						bestBuyPrice = orderbook.Bids[b].Price
					} else if bestBuyPrice < orderbook.Bids[b].Price {
						bestBuyPrice = orderbook.Bids[b].Price
					} else {
						continue
					}
				}

				fairPrice := (bestBuyPrice + bestSellPrice) / 2
				// exchangeTotalVolume := decimal.Sum(totalAskVolume, totalBidVolume)
				if result[fieldName].Price == 0 && result[fieldName].Volume == 0 {
					result[fieldName] = PriceReference{
						Exchange:  orderbook.Exchange,
						AssetType: orderbook.Asset.String(),
						isFuture:  orderbook.Asset.IsFutures(),
						Symbol:    orderbook.Pair.String(),
						Price:     fairPrice,
						Volume:    totalVolumeFloat,
					}
				} else if result[fieldName].Volume < totalVolumeFloat {
					result[fieldName] = PriceReference{
						Exchange:  orderbook.Exchange,
						AssetType: orderbook.Asset.String(),
						isFuture:  orderbook.Asset.IsFutures(),
						Symbol:    orderbook.Pair.String(),
						Price:     fairPrice,
						Volume:    totalVolumeFloat,
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
	if !atomic.CompareAndSwapInt32(&m.ProcessingOrder, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&m.ProcessingOrder, 0)
	fairPrices, err := m.GetFairPrice()
	if err != nil {
		log.Printf("error when get fair prices: %+v", err)
		return
	}
	log.Printf("%+v\n", fairPrices)
	for _, value := range fairPrices {
		if !strings.Contains(value.Symbol, "USDT") {
			continue
		}

		pair, err := currency.NewPairFromString(value.Symbol)
		if err != nil {
			log.Printf("error converting fair price symbol: %+v", err)
			continue
		}

		ccxPair := currency.NewPairWithDelimiter(pair.Base.String(), "USDT", "-")

		createdOrders, err := model.GetOrdersRedis(context.Background(),
			&order.Filter{Exchange: fixengine.CCX, Pair: ccxPair, Status: order.New},
			&order.Filter{},
		)

		if err != nil {
			log.Printf("error getting created order: %+v", err)
			continue
		}

		if len(createdOrders) == 0 {
			bidPriceLeves := GeneratePriceLevels(value.Price, "bid")
			for b := range bidPriceLeves {
				reqOrder := order.Detail{
					Exchange:  fixengine.CCX,
					AssetType: asset.Futures,
					Side:      order.Buy,
					Type:      order.Limit,
					Pair:      ccxPair,
					Price:     bidPriceLeves[b],
					Amount:    float64(b + 1),
				}
				if err := m.FixEngine.NewOrderSingle(reqOrder); err != nil {
					log.Printf("error when sent new order request: %+v", err)
					continue
				}
			}

			askPriceLeves := GeneratePriceLevels(value.Price, "ask")
			for b := range askPriceLeves {
				reqOrder := order.Detail{
					Exchange:  fixengine.CCX,
					AssetType: asset.Futures,
					Side:      order.Sell,
					Type:      order.Limit,
					Pair:      ccxPair,
					Price:     askPriceLeves[b],
					Amount:    float64(b + 1),
				}
				if err := m.FixEngine.NewOrderSingle(reqOrder); err != nil {
					log.Printf("error when sent new order request: %+v", err)
					continue
				}
			}
			continue
		}

		for i := range createdOrders {
			if createdOrders[i].Status.IsInactive() {
				continue
			}

			decimalCoef := m.GetRelevantPrice(value.Price)
			switch createdOrders[i].Amount {
			case 1:
				if (createdOrders[i].Price-value.Price) > (2*decimalCoef) || (createdOrders[i].Price-value.Price) < (-2*decimalCoef) {
					if err := m.FixEngine.CancelOrder(createdOrders[i]); err != nil {
						log.Printf("error when cancelling order: %+v", err)
						continue
					}
					continue
				}
				continue
			case 2:
				if (createdOrders[i].Price-value.Price) > (3*decimalCoef) || (createdOrders[i].Price-value.Price) < (-3*decimalCoef) {
					if err := m.FixEngine.CancelOrder(createdOrders[i]); err != nil {
						log.Printf("error when cancelling order: %+v", err)
						continue
					}
					continue
				}
				continue
			case 3:
				if (createdOrders[i].Price-value.Price) > (4*decimalCoef) || (createdOrders[i].Price-value.Price) < (-4*decimalCoef) {
					if err := m.FixEngine.CancelOrder(createdOrders[i]); err != nil {
						log.Printf("error when cancelling order: %+v", err)
						continue
					}
					continue
				}
				continue
			case 4:
				if (createdOrders[i].Price-value.Price) > (5*decimalCoef) || (createdOrders[i].Price-value.Price) < (-5*decimalCoef) {
					if err := m.FixEngine.CancelOrder(createdOrders[i]); err != nil {
						log.Printf("error when cancelling order: %+v", err)
						continue
					}
					continue
				}
				continue
			default:
				if (createdOrders[i].Price-value.Price) > (6*decimalCoef) || (createdOrders[i].Price-value.Price) < (-6*decimalCoef) {
					if err := m.FixEngine.CancelOrder(createdOrders[i]); err != nil {
						log.Printf("error when cancelling order: %+v", err)
						continue
					}
					continue
				}
				continue
			}
		}

		// bidPriceLeves := GeneratePriceLevels(value.Price, "bid")
		// for b := range bidPriceLeves {
		// 	reqOrder := order.Detail{
		// 		Exchange:  fixengine.CCX,
		// 		AssetType: asset.Futures,
		// 		Side:      order.Buy,
		// 		Type:      order.Limit,
		// 		Pair:      ccxPair,
		// 		Price:     bidPriceLeves[b],
		// 		Amount:    float64(b + 1),
		// 	}
		// 	if err := m.FixEngine.NewOrderSingle(reqOrder); err != nil {
		// 		log.Printf("error when sent new order request: %+v", err)
		// 		continue
		// 	}
		// }

		// askPriceLeves := GeneratePriceLevels(value.Price, "ask")
		// for b := range askPriceLeves {
		// 	reqOrder := order.Detail{
		// 		Exchange:  fixengine.CCX,
		// 		AssetType: asset.Futures,
		// 		Side:      order.Sell,
		// 		Type:      order.Limit,
		// 		Pair:      ccxPair,
		// 		Price:     askPriceLeves[b],
		// 		Amount:    float64(b + 1),
		// 	}
		// 	if err := m.FixEngine.NewOrderSingle(reqOrder); err != nil {
		// 		log.Printf("error when sent new order request: %+v", err)
		// 		continue
		// 	}
		// }
		continue
	}
}

func (m *MarketMaker) ShutdownRoutine() {
	existingOrders, err := model.GetOrdersRedis(context.Background(),
		&order.Filter{Exchange: fixengine.CCX, Status: order.New},
		nil,
	)
	if err != nil {
		log.Printf("error when try to shutdown market maker: %+v", err)
		return
	}

	if len(existingOrders) == 0 {
		return
	}

	for i := range existingOrders {
		if err := m.FixEngine.CancelOrder(existingOrders[i]); err != nil {
			log.Printf("error when shutting down market maker: %+v", err)
			return
		}
	}
}

func GeneratePriceLevels(price float64, side string) []float64 {
	priceDepth := make([]float64, 5)
	side = strings.ToUpper(side)
	decimals := countDecimals(price)
	switch decimals {
	case 0:
		for i := range priceDepth {
			if side == "BID" {
				price -= 2
			} else {
				price += 2
			}
			priceDepth[i] = decimal.NewFromFloatWithExponent(price, -2).InexactFloat64()
		}
	case 1:
		for i := range priceDepth {
			if side == "BID" {
				price -= 0.2
			} else {
				price += 0.2
			}
			priceDepth[i] = decimal.NewFromFloatWithExponent(price, -2).InexactFloat64()
		}
	default:
		for i := range priceDepth {
			if side == "BID" {
				price -= 0.02
			} else {
				price += 0.02
			}
			priceDepth[i] = decimal.NewFromFloatWithExponent(price, -2).InexactFloat64()
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

func countPrice(number float64) int {
	numbrStr := strconv.FormatFloat(number, 'f', -1, 64)
	parts := strings.Split(numbrStr, ".")
	return len(parts[0])
}

func (m *MarketMaker) GetNotionalAmout(price float64) float64 {
	priceLength := countPrice(price)
	switch priceLength {
	case 1, 2:
		return 1
	case 3:
		return 0.1
	default:
		return 0.01
	}
}
func (m *MarketMaker) GetRelevantPrice(price float64) float64 {
	decimals := countDecimals(price)
	switch decimals {
	case 0:
		return 1
	case 1:
		return 0.1
	default:
		return 0.01
	}
}
