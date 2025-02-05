package main

import (
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
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
	tempBPStore = make(map[string]map[order.Side]BestPrice)
	BPMutex     sync.Mutex
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

type BestPrice struct {
	Exchange  string
	AssetType string
	Symbol    string
	Side      order.Side
	Price     float64
	Volume    float64
}

func saveBestPrice(bestPrice BestPrice) {
	BPMutex.Lock()
	if tempBPStore[bestPrice.Symbol] == nil {
		tempBPStore[bestPrice.Symbol] = make(map[order.Side]BestPrice)
	}
	tempBPStore[bestPrice.Symbol][bestPrice.Side] = bestPrice
	BPMutex.Unlock()
}

func GetBestPrice(symbol string) map[order.Side]BestPrice {
	BPMutex.Lock()
	defer BPMutex.Unlock()

	bestMap, exist := tempBPStore[symbol]
	if !exist {
		return nil
	}
	return bestMap
}

func updateBestPrice(bestPrice BestPrice) error {
	BPMutex.Lock()
	defer BPMutex.Unlock()

	_, exist := tempBPStore[bestPrice.Symbol]
	if !exist {
		return errors.New("unavailable best price")
	}

	tempBPStore[bestPrice.Symbol][bestPrice.Side] = bestPrice
	return nil
}

func BestPriceProcess(ticks *ticker.Price, symbol string) error {
	if ticks == nil {
		return errors.New("ticker is nil")
	}

	if symbol == "" {
		return errors.New("invalid symbol")
	}

	bestPrice := GetBestPrice(symbol)
	bidBestPrice := BestPrice{
		Exchange:  ticks.ExchangeName,
		AssetType: ticks.AssetType.String(),
		Symbol:    symbol,
		Side:      order.Buy,
		Price:     ticks.Bid,
		Volume:    ticks.Volume,
	}
	askBestPrice := BestPrice{
		Exchange:  ticks.ExchangeName,
		AssetType: ticks.AssetType.String(),
		Symbol:    symbol,
		Side:      order.Sell,
		Price:     ticks.Ask,
		Volume:    ticks.Volume,
	}

	if bestPrice == nil {
		saveBestPrice(askBestPrice)
		saveBestPrice(bidBestPrice)
	}

	if bestPrice[bidBestPrice.Side].Price > bidBestPrice.Price && bidBestPrice.Price > 0 {
		if err := updateBestPrice(bidBestPrice); err != nil {
			return err
		}
	}

	if bestPrice[askBestPrice.Side].Price < askBestPrice.Price {
		if err := updateBestPrice(askBestPrice); err != nil {
			return err
		}
	}
	return nil
}
