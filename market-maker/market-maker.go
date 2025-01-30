package main

import (
	"context"
	"errors"
	"log"
	"math"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/market-maker/fixengine"
	model "github.com/thrasher-corp/gocryptotrader/market-maker/models"
)

type PriceReference struct {
	Exchange           string
	AssetType          string
	isFuture           bool
	Symbol             string
	Price              float64
	Volume             float64
	ContractMultiplier float64
	PriceMultiplier    float64
}

var (
	tempPRStore = make(map[string]PriceReference)
	PRMutex     sync.Mutex
)

func SavePriceReference(priceReference PriceReference) {
	PRMutex.Lock()
	tempPRStore[priceReference.Symbol] = priceReference
	PRMutex.Unlock()
}

func CheckPriceReference(symbol string) bool {
	PRMutex.Lock()
	defer PRMutex.Unlock()

	_, exist := tempPRStore[symbol]
	return exist
}

func GetPriceReference(symbol string) *PriceReference {
	PRMutex.Lock()
	defer PRMutex.Unlock()

	priceReference, exist := tempPRStore[symbol]
	if !exist {
		return nil
	}

	return &priceReference
}

func UpdatePriceReference(priceReference PriceReference) *PriceReference {
	PRMutex.Lock()
	defer PRMutex.Unlock()

	_, exist := tempPRStore[priceReference.Symbol]
	if !exist {
		return nil
	}
	tempPRStore[priceReference.Symbol] = priceReference

	return &priceReference
}

func ClearPRStore() {
	PRMutex.Lock()
	defer PRMutex.Unlock()

	for key := range tempPRStore {
		delete(tempPRStore, key)
	}
}

type MarketMaker struct {
	ProcessingOrder int32
	FetchTicker     int32
	FixEngine       *fixengine.FixEngine
	ExchangeManager *ExchangeManager
	SocketManager   *websocketRoutineManager
	PairFormatter   *currency.PairFormat
	Shutdown        chan struct{}
}

func NewMarketMaker(exchManager *ExchangeManager, eventRoutine *websocketRoutineManager) (*MarketMaker, error) {
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
	marketMaker.SocketManager = eventRoutine
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
	m.SocketManager.registerWebsocketDataHandler(m.WsDataHandler, false)
	m.ShutdownRoutine()
	go m.run()
	return nil
}

func (m *MarketMaker) Stop() {
	m.ShutdownRoutine()
	m.FixEngine.Stop()
	ClearPRStore()
	m.Shutdown <- struct{}{}
	close(m.Shutdown)
	return
}

func (m *MarketMaker) run() {
	m.PlaceOrder()
	ticker := time.NewTicker(time.Millisecond * 500)
	defer ticker.Stop()

	for {
		select {
		case <-m.Shutdown:
			ticker.Stop()
			return
		case <-ticker.C:
			go m.GetFairPrice()
			go m.PlaceOrder()
		}
	}
}

func (m *MarketMaker) GetFairPrice() {
	if !atomic.CompareAndSwapInt32(&m.FetchTicker, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&m.FetchTicker, 0)
	exchanges, err := m.ExchangeManager.GetExchanges()
	if err != nil {
		log.Printf("error when getting exchanges: %+v", err)
		return
	}

	if len(exchanges) == 0 {
		return
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

			for z := range ccxPairs {
				if enabledAssets[y] != asset.Spot {
					continue
				}

				priceTicker, err := exchanges[x].UpdateTicker(context.Background(), ccxPairs[z].Pair, enabledAssets[y])
				if err != nil {
					if strings.Contains(err.Error(), "400") || strings.Contains(err.Error(), "not found") {
						continue
					}
					log.Printf("Error when fetch %s order book from %s %s: %+v", ccxPairs[z].Pair.String(), exchanges[x].GetName(), enabledAssets[y].String(), err)
					continue
				}

				symbol := m.PairFormatter.Format(priceTicker.Pair)
				quoteCurrency := strings.Split(priceTicker.Pair.Quote.String(), "-")[0]
				var fieldName string
				switch priceTicker.AssetType.IsFutures() {
				case true:
					fieldName = priceTicker.Pair.Base.String() + "-" + quoteCurrency
				case false:
					fieldName = symbol
				}

				var fairPrice float64
				if priceTicker.Bid != 0 && priceTicker.Ask != 0 {
					fairPrice = math.Abs((priceTicker.Bid + priceTicker.Ask) / 2)
				} else if priceTicker.Bid != 0 && priceTicker.Ask == 0 {
					fairPrice = priceTicker.Bid
				} else if priceTicker.Ask != 0 && priceTicker.Bid == 0 {
					fairPrice = priceTicker.Ask
				}
				exchangeTotalVolume := math.Abs(priceTicker.Volume)
				result := GetPriceReference(fieldName)
				if result == nil {
					result = &PriceReference{
						Exchange:           priceTicker.ExchangeName,
						AssetType:          priceTicker.AssetType.String(),
						isFuture:           priceTicker.AssetType.IsFutures(),
						Symbol:             symbol,
						Price:              fairPrice,
						Volume:             exchangeTotalVolume,
						PriceMultiplier:    ccxPairs[z].PriceMultiplier,
						ContractMultiplier: ccxPairs[z].ContractMultiplier,
					}
					SavePriceReference(*result)
				} else if math.Abs(result.Volume) < exchangeTotalVolume {
					result = &PriceReference{
						Exchange:           priceTicker.ExchangeName,
						AssetType:          priceTicker.AssetType.String(),
						isFuture:           priceTicker.AssetType.IsFutures(),
						Symbol:             symbol,
						Price:              fairPrice,
						Volume:             exchangeTotalVolume,
						PriceMultiplier:    ccxPairs[z].PriceMultiplier,
						ContractMultiplier: ccxPairs[z].ContractMultiplier,
					}
					UpdatePriceReference(*result)
				} else {
					continue
				}
			}
		}
	}
}

func (m *MarketMaker) PlaceOrder() {
	if !atomic.CompareAndSwapInt32(&m.ProcessingOrder, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&m.ProcessingOrder, 0)
	fairPrices := tempPRStore

	if len(fairPrices) == 0 {
		return
	}
	log.Printf("fairPrices: %+v", fairPrices)
FairPricesLoop:
	for _, value := range fairPrices {
		if !strings.Contains(value.Symbol, "USDT") {
			continue
		}

		pair, err := currency.NewPairFromString(value.Symbol)
		if err != nil {
			log.Printf("error converting fair price symbol: %+v", err)
			continue
		}

		ccxPair := currency.NewPairWithDelimiter(pair.Base.String(), "USD", "-")

		createdOrders, err := model.GetOrdersRedis(context.Background(),
			&order.Filter{Exchange: fixengine.CCX, Pair: ccxPair, Status: order.New},
			nil,
		)

		if err != nil {
			log.Printf("error getting created order: %+v", err)
			continue
		}

		if len(createdOrders) == 0 {
			bidPriceLeves := GeneratePriceLevels(value.Price, value.PriceMultiplier, "bid")
			for b := range bidPriceLeves {
				reqOrder := order.Detail{
					Exchange:  fixengine.CCX,
					AssetType: asset.Futures,
					Side:      order.Buy,
					Type:      order.Limit,
					Pair:      ccxPair,
					Price:     bidPriceLeves[b],
					Amount:    quantityLevels[b%len(quantityLevels)], // use config supplied quantity level that base on book depth and prevent out of range error
				}

				if err := m.FixEngine.NewOrderSingle(reqOrder); err != nil {
					log.Printf("error when sent new order request: %+v", err)
					continue
				}
			}

			askPriceLeves := GeneratePriceLevels(value.Price, value.PriceMultiplier, "ask")
			for b := range askPriceLeves {
				reqOrder := order.Detail{
					Exchange:  fixengine.CCX,
					AssetType: asset.Futures,
					Side:      order.Sell,
					Type:      order.Limit,
					Pair:      ccxPair,
					Price:     askPriceLeves[b],
					Amount:    quantityLevels[b%len(quantityLevels)],
				}
				if err := m.FixEngine.NewOrderSingle(reqOrder); err != nil {
					log.Printf("error when sent new order request: %+v", err)
					continue
				}
			}
			continue
		}

	CancelOrderLoop:
		for i := range createdOrders {
			if createdOrders[i].Status.IsInactive() {
				continue
			}

			if !m.CheckPriceDifference(value.Price, createdOrders[i], value.PriceMultiplier) {
				// if err := m.CancelAllOrders(createdOrders); err != nil {
				// 	log.Printf("error when cancelling orders: %+v", err)
				// 	continue
				// }
				if err := m.ModifyOrders(createdOrders, value); err != nil {
					log.Printf("error when modifying orders: %+v", err)
					continue
				}
				log.Printf("price changed for %s", createdOrders[i].Pair.Base.String())
				break CancelOrderLoop
			}
			log.Printf("price not change for %s", createdOrders[i].Pair.Base.String())
			continue FairPricesLoop
		}

		// bidPriceLeves := GeneratePriceLevels(value.Price, value.PriceMultiplier, "bid")
		// for b := range bidPriceLeves {
		// 	reqOrder := order.Detail{
		// 		Exchange:  fixengine.CCX,
		// 		AssetType: asset.Futures,
		// 		Side:      order.Buy,
		// 		Type:      order.Limit,
		// 		Pair:      ccxPair,
		// 		Price:     bidPriceLeves[b],
		// 		Amount:    quantityLevels[b%len(quantityLevels)],
		// 	}
		// 	if err := m.FixEngine.NewOrderSingle(reqOrder); err != nil {
		// 		log.Printf("error when sent new order request: %+v", err)
		// 		continue
		// 	}
		// }

		// askPriceLeves := GeneratePriceLevels(value.Price, value.PriceMultiplier, "ask")
		// for b := range askPriceLeves {
		// 	reqOrder := order.Detail{
		// 		Exchange:  fixengine.CCX,
		// 		AssetType: asset.Futures,
		// 		Side:      order.Sell,
		// 		Type:      order.Limit,
		// 		Pair:      ccxPair,
		// 		Price:     askPriceLeves[b],
		// 		Amount:    quantityLevels[b%len(quantityLevels)],
		// 	}
		// 	if err := m.FixEngine.NewOrderSingle(reqOrder); err != nil {
		// 		log.Printf("error when sent new order request: %+v", err)
		// 		continue
		// 	}
		// }
		continue
	}
}

func (m *MarketMaker) CheckPriceDifference(fairPrice float64, orderDetail order.Detail, priceMultiplier float64) bool {
	var allowedDifference float64
	fairPrice = decimal.NewFromFloatWithExponent(fairPrice, -2).InexactFloat64()
	// if fairPrice > 999 {
	// 	priceMultiplier = 1
	// }
	switch orderDetail.Amount {
	case quantityLevel1:
		allowedDifference = priceLevel1 * priceMultiplier
	case quantityLevel2:
		allowedDifference = priceLevel2 * priceMultiplier
	case quantityLevel3:
		allowedDifference = priceLevel3 * priceMultiplier
	case quantityLevel4:
		allowedDifference = priceLevel4 * priceMultiplier
	default:
		allowedDifference = priceLevel5 * priceMultiplier
	}

	pricedifference := orderDetail.Price - fairPrice
	switch orderDetail.Side {
	case order.Buy:
		log.Printf("price difference: %f", pricedifference)
		log.Printf("allowed diffence: %f", -allowedDifference)
		log.Printf("gap between price difference and allowed difference: %f", math.Abs(pricedifference - -allowedDifference))
		return math.Abs(pricedifference - -allowedDifference) <= priceGapTolerance
	default:
		log.Printf("price difference: %f", pricedifference)
		log.Printf("allowed diffence: %f", allowedDifference)
		log.Printf("gap between price difference and allwed difference: %f", math.Abs(pricedifference-allowedDifference))
		return math.Abs(pricedifference-allowedDifference) <= priceGapTolerance
	}
}

func (m *MarketMaker) CancelAllOrders(orders []order.Detail) error {
	for i := range orders {
		if err := m.FixEngine.CancelOrder(orders[i]); err != nil {
			return err
		}
	}
	return nil
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

func (m *MarketMaker) WsDataHandler(exchName string, data interface{}) error {
	if m == nil {
		return nil
	}

	switch d := data.(type) {
	case *ticker.Price:
		if d.AssetType != asset.Spot || d.Pair.Quote.String() != "USDT" {
			return nil
		}

		ccxPair, err := model.GetPair(context.Background(), d.Pair.Base.String())
		if err != nil {
			return err
		}

		if ccxPair == nil {
			return nil
		}

		symbol := m.PairFormatter.Format(d.Pair)
		fairPrice := GetPriceReference(symbol)
		if fairPrice == nil {
			fairPrice = &PriceReference{
				Exchange:           exchName,
				AssetType:          d.AssetType.String(),
				isFuture:           d.AssetType.IsFutures(),
				Symbol:             symbol,
				Volume:             math.Abs(d.Volume),
				ContractMultiplier: ccxPair.ContractMultiplier,
				PriceMultiplier:    ccxPair.PriceIncrement,
			}
			if d.Ask != 0 && d.Bid != 0 {
				fairPrice.Price = math.Abs((d.Ask + d.Bid) / 2)
			} else if d.Ask != 0 && d.Bid == 0 {
				fairPrice.Price = d.Ask
			} else if d.Bid != 0 && d.Ask == 0 {
				fairPrice.Price = d.Bid
			}
			SavePriceReference(*fairPrice)
			return nil
		} else if math.Abs(fairPrice.Volume) < math.Abs(d.Volume) {
			fairPrice = &PriceReference{
				Exchange:           exchName,
				AssetType:          d.AssetType.String(),
				isFuture:           d.AssetType.IsFutures(),
				Symbol:             symbol,
				Volume:             math.Abs(d.Volume),
				ContractMultiplier: ccxPair.ContractMultiplier,
				PriceMultiplier:    ccxPair.PriceIncrement,
			}
			if d.Ask != 0 && d.Bid != 0 {
				fairPrice.Price = math.Abs((d.Ask + d.Bid) / 2)
			} else if d.Ask != 0 && d.Bid == 0 {
				fairPrice.Price = d.Ask
			} else if d.Bid != 0 && d.Ask == 0 {
				fairPrice.Price = d.Bid
			}
			UpdatePriceReference(*fairPrice)
		}

		return nil
	default:
	}
	return nil
}

func (m *MarketMaker) ModifyOrders(orders []order.Detail, fairPrice PriceReference) error {
	buyPrices := GeneratePriceLevels(fairPrice.Price, fairPrice.PriceMultiplier, "bid")
BuyOrdersLoop:
	for b := range buyPrices {
		for i := range orders {
			if orders[i].Side != order.Buy {
				continue
			}
			log.Printf("unmodified order: %+v", orders[i])
			orders[i].Price = buyPrices[b]
			orders[i].Amount = quantityLevels[b%len(quantityLevels)]

			if err := m.FixEngine.CancelReplaceOrder(orders[i]); err != nil {
				return err
			}
			log.Printf("poped order: %+v", orders[i])
			orders = append(orders[:i], orders[i+1:]...)
			log.Printf("after poped order: %+v", orders)
			continue BuyOrdersLoop
		}
	}

	sellPrices := GeneratePriceLevels(fairPrice.Price, fairPrice.PriceMultiplier, "ask")
SellOrdersLoop:
	for a := range sellPrices {
		for i := range orders {
			if orders[i].Side != order.Sell {
				continue
			}
			log.Printf("unmodified order: %+v", orders[i])
			orders[i].Price = buyPrices[a]
			orders[i].Amount = quantityLevels[a%len(quantityLevels)]
			// modify existing order
			if err := m.FixEngine.CancelReplaceOrder(orders[i]); err != nil {
				return err
			}
			log.Printf("poped order: %+v", orders[i])
			orders = append(orders[:i], orders[i+1:]...) // remove modified order from the list
			log.Printf("after poped order: %+v", orders)
			continue SellOrdersLoop
		}
	}
	return nil
}

func GeneratePriceLevels(price, priceMultiplier float64, side string) []float64 {
	priceDepth := make([]float64, priceLevelDepth)
	side = strings.ToUpper(side)

	// if price > 999 {
	// 	priceMultiplier = 1
	// }

	for i := range priceDepth {
		priceLevel := priceLevels[i%len(priceLevels)] * priceMultiplier

		if side == "BID" {
			priceDepth[i] = price - priceLevel
		} else {
			priceDepth[i] = price + priceLevel
		}
	}
	return priceDepth
}
