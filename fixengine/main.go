package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/core"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/signaler"
)

func main() {
	// Handle flags
	var settings Settings
	versionFlag := flag.Bool("version", false, "retrieves current GoCryptoTrader version")

	// Core settings
	flag.StringVar(&settings.ConfigFile, "config", config.DefaultFilePath(), "config file to load")
	flag.StringVar(&settings.DataDir, "datadir", common.GetDefaultDataDir(runtime.GOOS), "default data directory for GoCryptoTrader files")
	flag.BoolVar(&settings.EnableAllExchanges, "enableallexchanges", false, "enables all exchanges")
	flag.BoolVar(&settings.EnableAllPairs, "enableallpairs", false, "enables all pairs for enabled exchanges")
	flag.BoolVar(&settings.Verbose, "verbose", false, "increases logging verbosity for GoCryptoTrader")
	flag.BoolVar(&settings.EnableFuturesTracking, "enablefuturestracking", true, "tracks futures orders PNL is supported by the exchange")
	flag.BoolVar(&settings.EnableExchangeSyncManager, "syncmanager", false, "enables to exchange sync manager")
	flag.BoolVar(&settings.EnableWebsocketRoutine, "websocketroutine", true, "enables the websocket routine for all loaded exchanges")
	flag.BoolVar(&settings.EnableConnectivityMonitor, "connectivitymonitor", true, "enables the connectivity monitor")
	flag.BoolVar(&settings.EnableDispatcher, "dispatch", false, "enables the dispatch system")

	// Exchange syncer settings
	flag.BoolVar(&settings.EnableTickerSyncing, "tickersync", true, "enables ticker syncing for all enabled exchanges")
	flag.BoolVar(&settings.EnableOrderbookSyncing, "orderbooksync", true, "enables orderbook syncing for all enabled exchanges")
	flag.BoolVar(&settings.EnableTradeSyncing, "tradesync", false, "enables trade syncing for all enabled exchanges")
	flag.IntVar(&settings.SyncWorkersCount, "syncworkers", DefaultSyncerWorkers, "the amount of workers (goroutines) to use for syncing exchange data")
	flag.BoolVar(&settings.SyncContinuously, "synccontinuously", true, "whether to sync exchange data continuously (ticker, orderbook and trade history info")
	flag.DurationVar(&settings.SyncTimeoutREST, "synctimeoutrest", DefaultSyncerTimeoutREST,
		"the amount of time before the syncer will switch from rest protocol to the streaming protocol (e.g. from REST to websocket)")
	flag.DurationVar(&settings.SyncTimeoutWebsocket, "synctimeoutwebsocket", DefaultSyncerTimeoutWebsocket,
		"the amount of time before the syncer will switch from the websocket protocol to REST protocol (e.g. from websocket to REST)")
	flag.BoolVar(&settings.EnableFIXEngine, "fixengine", true, "enables FIX Engine")
	flag.Parse()

	if *versionFlag {
		fmt.Print(core.Version(true))
		os.Exit(0)
	}

	fmt.Println(core.Version(false))

	var err error
	settings.CheckParamInteraction = true

	// collect flags
	flagSet := make(map[string]bool)
	// Stores the set flags
	flag.Visit(func(f *flag.Flag) { flagSet[f.Name] = true })
	if !flagSet["config"] {
		// If config file is not explicitly set, fall back to default path resolution
		settings.ConfigFile = ""
	}

	settings.Shutdown = make(chan struct{})
	engine, err := NewFromSettings(&settings, flagSet)
	if engine == nil || err != nil {
		log.Fatalf("Unable to initialise FIX engine. Error: %s\n", err)
	}
	config.SetConfig(engine.Config)

	engine.Settings.PrintLoadedSettings()

	if err = engine.Start(); err != nil {
		errClose := gctlog.CloseLogger()
		if errClose != nil {
			log.Printf("Unable to close logger. Error: %s\n", errClose)
		}
		log.Fatalf("Unable to start FIX engine. Error: %s\n", err)
	}

	go waitForInterrupt(settings.Shutdown)
	<-settings.Shutdown
	engine.Stop()
}

func waitForInterrupt(waiter chan<- struct{}) {
	interrupt := signaler.WaitForInterrupt()
	gctlog.Infof(gctlog.Global, "Captured %v, shutdown requested.\n", interrupt)
	waiter <- struct{}{}
}
