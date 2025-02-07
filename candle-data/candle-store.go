package main

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/log"
)

type CandleStore struct {
	fetchInstrumentProcess int32
	fetchCandleprocess     int32
	ExchangeManager        *ExchangeManager
	WebsocketHandler       *websocketRoutineManager
	PairFormatter          *currency.PairFormat
	Shutdown               chan struct{}
}

func NewCandleStore(exch *ExchangeManager, websocketHandler *websocketRoutineManager) (*CandleStore, error) {
	if exch == nil {
		return nil, errNilExchangeManager
	}

	if websocketHandler == nil {
		return nil, errNilWebsocket
	}

	candleStore := CandleStore{
		PairFormatter: &currency.PairFormat{
			Uppercase: true,
			Delimiter: "-",
		},
		Shutdown: make(chan struct{}),
	}

	candleStore.ExchangeManager = exch
	candleStore.WebsocketHandler = websocketHandler
	return &candleStore, nil
}

func (c *CandleStore) Start() {
	log.Infoln(log.Global, "Fetching instruments...")
	c.FetchInstruments()
	log.Infoln(log.Global, "Fetching candles data...")
	c.FetchHistoricCandles()
	c.PrintData()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			go c.FetchInstruments()
			go c.FetchHistoricCandles()
			go c.PrintData()
		case <-c.Shutdown:
			log.Infoln(log.Global, "candle data engine shutting down...")
			return
		}
	}
}

func (c *CandleStore) Stop() {
	c.Shutdown <- struct{}{}
	ClearInstrumentStore()
	ClearCandleStore()
	close(c.Shutdown)
	return
}

func (c *CandleStore) FetchInstruments() {
	if !atomic.CompareAndSwapInt32(&c.fetchInstrumentProcess, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&c.fetchInstrumentProcess, 0)

	exchanges, err := c.ExchangeManager.GetExchanges()
	if err != nil {
		log.Errorf(log.ExchangeSys, "Error getting available exchanges: %+v", err)
		return
	}

	if len(exchanges) == 0 {
		return
	}

	for x := range exchanges {
		if !exchanges[x].GetBase().Enabled {
			continue
		}

		enableAssets := exchanges[x].GetAssetTypes(true)
		if len(enableAssets) == 0 {
			continue
		}

		for y := range enableAssets {
			tradablePairs, err := exchanges[x].GetEnabledPairs(enableAssets[y])
			if err != nil {
				log.Errorf(log.ExchangeSys, "Error getting available pairs")
				continue
			}

			if len(tradablePairs) == 0 {
				return
			}

			for z := range tradablePairs {
				symbol := c.PairFormatter.Format(tradablePairs[z])
				existingInstrument := GetInstrument(symbol)
				if existingInstrument != nil {
					continue
				}
				instrument := Instrument{
					Exchange:  exchanges[x].GetName(),
					AssetType: enableAssets[y].String(),
					Symbol:    symbol,
				}
				SaveInstrument(instrument)
			}
		}
	}
}

func (c *CandleStore) FetchHistoricCandles() {
	if !atomic.CompareAndSwapInt32(&c.fetchCandleprocess, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&c.fetchCandleprocess, 0)

	instruments := instrumentStore
	if len(instruments) == 0 {
		return
	}

	for _, instrument := range instruments {
		exchange, err := c.ExchangeManager.GetExchangeByName(instrument.Exchange)
		if err != nil {
			log.Errorf(log.ExchangeSys, "error getting exchange: %+v", err)
			continue
		}

		asset, err := asset.New(instrument.AssetType)
		if err != nil {
			log.Errorf(log.ExchangeSys, "error converting asset: %+v", err)
			continue
		}

		pair, err := currency.NewPairFromString(instrument.Symbol)
		if err != nil {
			log.Errorf(log.Currency, "error converting symbol: %+v", err)
			continue
		}

		kline, err := exchange.GetHistoricCandles(context.TODO(), pair, asset, kline.OneHour, time.Now().Add(-1*time.Hour), time.Now())
		if err != nil {
			log.Errorf(log.ExchangeSys, "error fetch candle data: %+v", err)
			continue
		}
		if err := KlineProcess(kline, instrument.Symbol); err != nil {
			log.Errorf(log.Global, "error processing candle data: %+v", err)
			continue
		}
	}
}

func (c *CandleStore) PrintData() {
	instruments := instrumentStore
	candles := candleStore
	c.jsonOutput(instruments, "available instruments")
	c.jsonOutput(candles, "candles data")
}

func (c *CandleStore) jsonOutput(data interface{}, info string) {
	j, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		log.Errorf(log.Global, "error from json marshal: %+v", err)
		return
	}
	log.Infof(log.Global, "%s: \n%s", info, string(j))
}
