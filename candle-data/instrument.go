package main

import "sync"

type Instrument struct {
	Exchange  string `json:"exchange,omitempty"`
	AssetType string `json:"asset_type,omitempty"`
	Symbol    string `json:"symbol,omitempty"`
}

var (
	instrumentStore = make(map[string]Instrument)
	instrumentMutex sync.Mutex
)

func GetInstrument(symbol string) *Instrument {
	instrumentMutex.Lock()
	defer instrumentMutex.Unlock()

	instrument, exist := instrumentStore[symbol]
	if !exist {
		return nil
	}

	return &instrument
}

func SaveInstrument(instrument Instrument) {
	instrumentMutex.Lock()
	instrumentStore[instrument.Symbol] = instrument
	instrumentMutex.Unlock()
}

func ClearInstrumentStore() {
	instrumentMutex.Lock()
	defer instrumentMutex.Unlock()

	for key := range instrumentStore {
		delete(instrumentStore, key)
	}
}
