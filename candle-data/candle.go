package main

import (
	"errors"
	"sync"

	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
)

type CandleData struct {
	Time             int64   `json:"time,omitempty"`
	Open             float64 `json:"open,omitempty"`
	High             float64 `json:"high,omitempty"`
	Low              float64 `json:"low,omitempty"`
	Close            float64 `json:"close,omitempty"`
	Volume           float64 `json:"volume,omitempty"`
	ValidationIssues string  `json:"validation_issues,omitempty"`
}

type Candle struct {
	Exchange  string       `json:"exchange,omitempty"`
	AssetType string       `json:"asset_type,omitempty"`
	Symbol    string       `json:"symbol,omitempty"`
	Interval  string       `json:"interval,omitempty"`
	Candles   []CandleData `json:"candles,omitempty"`
}

var (
	candleStore = make(map[string]map[string]Candle)
	candleMutex sync.Mutex
)

func saveCandle(candle Candle) {
	candleMutex.Lock()
	if candleStore[candle.Symbol] == nil {
		candleStore[candle.Symbol] = make(map[string]Candle)
	}
	candleStore[candle.Symbol][candle.Interval] = candle
	candleMutex.Unlock()
}

func GetCandle(symbol string, interval *string) []Candle {
	candleMutex.Lock()
	defer candleMutex.Unlock()
	var candles []Candle
	if symbol == "" {
		return candles
	}

	candlesMap, exist := candleStore[symbol]
	if !exist {
		return candles
	}

	if interval != nil && *interval != "" {
		candles = make([]Candle, 1)
		candle, exist := candlesMap[*interval]
		if !exist {
			return candles
		}
		candles = append(candles, candle)
		return candles
	}

	for interval := range candlesMap {
		candles = append(candles, candlesMap[interval])
	}
	return candles
}

func ClearCandleStore() {
	candleMutex.Lock()
	defer candleMutex.Unlock()

	for symbol := range candleStore {
		delete(candleStore, symbol)
	}
}

func KlineProcess(kline *kline.Item, symbol string) error {
	if kline == nil {
		return errors.New("no data provided")
	}

	if symbol == "" {
		return errors.New("symbol was empty")
	}

	interval := kline.Interval.String()
	candles := GetCandle(symbol, &interval)

	if len(candles) == 0 {
		candle := Candle{
			Exchange:  kline.Exchange,
			AssetType: kline.Asset.String(),
			Symbol:    symbol,
			Interval:  interval,
		}

		if len(kline.Candles) == 0 {
			saveCandle(candle)
		}

		candleData := make([]CandleData, len(kline.Candles))
		for i := range kline.Candles {
			candleData[i] = CandleData{
				Time:             kline.Candles[i].Time.UnixMilli(),
				Open:             kline.Candles[i].Open,
				High:             kline.Candles[i].High,
				Low:              kline.Candles[i].Low,
				Close:            kline.Candles[i].Close,
				Volume:           kline.Candles[i].Volume,
				ValidationIssues: kline.Candles[i].ValidationIssues,
			}
		}
		candle.Candles = candleData
		saveCandle(candle)
	}

	for c := range candles {
		if candles[c].Interval == kline.Interval.String() {
			if len(kline.Candles) == 0 {
				continue
			}

			candleData := make([]CandleData, len(kline.Candles))
			for i := range kline.Candles {
				candleData[i] = CandleData{
					Time:             kline.Candles[i].Time.UnixMilli(),
					Open:             kline.Candles[i].Open,
					High:             kline.Candles[i].High,
					Low:              kline.Candles[i].Low,
					Close:            kline.Candles[i].Close,
					Volume:           kline.Candles[i].Volume,
					ValidationIssues: kline.Candles[i].ValidationIssues,
				}
			}
			candles[c].Candles = append(candles[c].Candles, candleData...)
			saveCandle(candles[c])
		}
	}
	return nil
}

func SyncDatabase() {
	if len(candleStore) == 0 {
		return
	}

}
