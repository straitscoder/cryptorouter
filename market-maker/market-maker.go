package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/market-maker/fixengine"
	model "github.com/thrasher-corp/gocryptotrader/market-maker/models"
)

type MarketMaker struct {
	ProcessingOrder      int32
	FetchTicker          int32
	processExchangeOrder int32
	FixEngine            *fixengine.FixEngine
	ExchangeManager      *ExchangeManager
	SocketManager        *websocketRoutineManager
	PairFormatter        *currency.PairFormat
	Shutdown             chan struct{}
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
	marketMaker.FixEngine = fixengine.NewFixEngine()
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
	m.Shutdown <- struct{}{}
	ClearPRStore()
	ClearBPStore()
	m.ShutdownRoutine()
	m.FixEngine.Stop()
	// if err := model.DeleteOrders(context.Background()); err != nil {
	// 	log.Printf("failed to delete orders: %+v", err)
	// }
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
			go m.CreateCounterOrder()
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
				if err := BestPriceProcess(priceTicker, fieldName); err != nil {
					log.Printf("error when best price processing: %+v", err)
				}
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
	// bestPrices := tempBPStore

	if len(fairPrices) == 0 {
		return
	}
	// log.Printf("fairPrices: %+v", fairPrices)
	// log.Printf("best prices: %+v", bestPrices)
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
		// log.Printf("length of created orders for %s: %d", value.Symbol, len(createdOrders))
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
			continue FairPricesLoop
		}

	ModifyOrderLoop:
		for i := range createdOrders {
			if createdOrders[i].Status.IsInactive() {
				continue
			}

			if !m.CheckPriceDifference(value.Price, createdOrders[i], value.PriceMultiplier) || len(createdOrders) != priceLevelDepth*2 {
				if err := m.CancelAllOrders(createdOrders); err != nil {
					log.Printf("error when cancelling orders: %+v", err)
					continue
				}
				// if err := m.ModifyOrders(createdOrders, value); err != nil {
				// 	log.Printf("error when modifying orders: %+v", err)
				// 	continue
				// }
				log.Printf("price changed for %s", createdOrders[i].Pair.Base.String())
				break ModifyOrderLoop
			}
			log.Printf("price not change for %s", createdOrders[i].Pair.Base.String())
			continue FairPricesLoop
		}

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
		// log.Printf("price difference: %f", pricedifference)
		// log.Printf("allowed diffence: %f", -allowedDifference)
		// log.Printf("gap between price difference and allowed difference: %f", math.Abs(pricedifference - -allowedDifference))
		return math.Abs(pricedifference - -allowedDifference) <= priceGapTolerance
	case order.Sell:
		// log.Printf("price difference: %f", pricedifference)
		// log.Printf("allowed diffence: %f", allowedDifference)
		// log.Printf("gap between price difference and allowed difference: %f", math.Abs(pricedifference-allowedDifference))
		return math.Abs(pricedifference-allowedDifference) <= priceGapTolerance
	default:
		log.Printf("invalid side: %+v", orderDetail)
		return true
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
		if err := BestPriceProcess(d, symbol); err != nil {
			return err
		}
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
	case *order.Detail:
		if d == nil {
			return nil
		}

		if err := model.UpdateOrCreateOrder(*d, fmt.Sprintf("update order from %s websocket", d.Exchange)); err != nil {
			return err
		}
		log.Printf("created order from external exchange: %+v", d)
	default:
	}
	return nil
}

func (m *MarketMaker) ModifyOrders(orders []order.Detail, fairPrice PriceReference) error {
	var modifiedOrder []order.Detail
	priceMap := map[order.Side][]float64{
		order.Buy:  GeneratePriceLevels(fairPrice.Price, fairPrice.PriceMultiplier, "bid"),
		order.Sell: GeneratePriceLevels(fairPrice.Price, fairPrice.PriceMultiplier, "ask"),
	}

	for side, prices := range priceMap {
	PricesLoop:
		for x := range prices {
			if len(orders) != 0 {
				for y := range orders {
					if side != orders[y].Side {
						continue
					}

					orders[y].Price = prices[x]
					orders[y].Amount = quantityLevels[x%len(quantityLevels)]

					modifiedOrder = append(modifiedOrder, orders[y])
					orders = append(orders[:y], orders[y+1:]...)
					continue PricesLoop
				}
			}
			// log.Printf("triggered on price index %d of %s field", x, side.String())
			// create missing order
			pair, err := currency.NewPairFromString(fairPrice.Symbol)
			if err != nil {
				return err
			}
			reqOrder := order.Detail{
				Exchange:  fixengine.CCX,
				AssetType: asset.Futures,
				Side:      side,
				Type:      order.Limit,
				Pair:      pair,
				Price:     prices[x],
				Amount:    quantityLevels[x%len(quantityLevels)],
			}

			if err := m.FixEngine.NewOrderSingle(reqOrder); err != nil {
				return err
			}
			continue PricesLoop
		}
	}

	if len(modifiedOrder) == 0 {
		return nil
	}

	for i := range modifiedOrder {
		if err := m.FixEngine.CancelReplaceOrder(modifiedOrder[i]); err != nil {
			return err
		}
	}
	return nil
}

func (m *MarketMaker) CreateCounterOrder() {
	if !atomic.CompareAndSwapInt32(&m.processExchangeOrder, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&m.processExchangeOrder, 0)
	orderDetail, err := model.GetCounterOrderQueue(context.Background())
	if err != nil {
		log.Printf("error when get counter order queue: %+v", err)
		return
	}

	if orderDetail.OrderID == "" {
		return
	}

	existingOrder := model.GetOrderByClOrdID(orderDetail.ClientOrderID)
	if existingOrder.OrderID != "" {
		return
	}

	orderDetail.Pair = currency.NewPairWithDelimiter(orderDetail.Pair.Base.String(), "USDT", "-")
	symbol := m.PairFormatter.Format(orderDetail.Pair)
	priceReference := GetPriceReference(symbol)
	if priceReference == nil {
		log.Printf("no price reference provided for: %+v", orderDetail)
		return
	}

	exch, err := m.ExchangeManager.GetExchangeByName("okx")
	if err != nil {
		log.Printf("error when get exchange for: %+v", priceReference)
		return
	}

	a, err := asset.New(priceReference.AssetType)
	if err != nil {
		log.Printf("invalid asset: %+v", priceReference)
		return
	}

	err = exch.CheckOrderExecutionLimits(a, orderDetail.Pair, priceReference.Price, orderDetail.Amount, orderDetail.Type)
	if err != nil {
		log.Printf("Execution limit error %+v", err)
		return
	}

	err = exch.CanTradePair(orderDetail.Pair, a)
	if err != nil {
		log.Printf("Trade pair error: %+v", err)
		return
	}

	submiRequest := order.Submit{
		Exchange:      exch.GetName(),
		Type:          orderDetail.Type,
		AssetType:     a,
		Pair:          orderDetail.Pair,
		ClientOrderID: orderDetail.ClientOrderID,
		Price:         orderDetail.Price,
		Amount:        orderDetail.Amount,
		Side:          orderDetail.Side,
	}

	response, err := exch.SubmitOrder(context.TODO(), &submiRequest)
	if err != nil {
		log.Printf("Failed to submit this order: %+v", submiRequest)
		return
	}

	if response == nil {
		return
	}
	log.Printf("Submitted order: %+v", *response)

	internalId, _ := uuid.NewV4()
	willSaveORder, err := response.DeriveDetail(internalId)
	if err != nil {
		log.Printf("error when generate order detail: %+v", err)
		return
	}

	if err := model.UpdateOrCreateOrder(*willSaveORder, fmt.Sprintf("Hedging order from %s", willSaveORder.Exchange)); err != nil {
		log.Printf("error when save hedging order: %+v", err)
		return
	}
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
