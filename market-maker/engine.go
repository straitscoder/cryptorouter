package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/config"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/dispatch"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	gctlog "github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/utils"
)

// Settings stores engine params. Please define a settings struct for automatic
// display of instance settings. For example, if you define a struct named
// ManagerSettings, it will be displayed as a subheading "Manager Settings"
// and individual field names such as 'EnableManager' will be displayed
// as "Enable Manager: true/false".
type Settings struct {
	ConfigFile            string
	DataDir               string
	MigrationDir          string
	LogFile               string
	GoMaxProcs            int
	CheckParamInteraction bool
	EnabeTradingEngine    bool

	CoreSettings
	ExchangeSyncerSettings
	ExchangeTuningSettings

	// Main shutdown channel
	Shutdown chan struct{}
}

// CoreSettings defines settings related to core engine operations
type CoreSettings struct {
	EnableDryRun                bool
	EnableAllExchanges          bool
	EnableAllPairs              bool
	EnableCoinmarketcapAnalysis bool
	EnablePortfolioManager      bool
	EnableDataHistoryManager    bool
	PortfolioManagerDelay       time.Duration
	EnableGRPC                  bool
	EnableGRPCProxy             bool
	EnableGRPCShutdown          bool
	EnableWebsocketRPC          bool
	EnableDeprecatedRPC         bool
	EnableCommsRelayer          bool
	EnableExchangeSyncManager   bool
	EnableDepositAddressManager bool
	EnableEventManager          bool
	EnableOrderManager          bool
	EnableConnectivityMonitor   bool
	EnableDatabaseManager       bool
	EnableGCTScriptManager      bool
	EnableNTPClient             bool
	EnableWebsocketRoutine      bool
	EnableCurrencyStateManager  bool
	EventManagerDelay           time.Duration
	EnableFuturesTracking       bool
	Verbose                     bool
	EnableDispatcher            bool
	DispatchMaxWorkerAmount     int
	DispatchJobsLimit           int
}

// ExchangeSyncerSettings defines settings for the exchange pair synchronisation
type ExchangeSyncerSettings struct {
	EnableTickerSyncing    bool
	EnableOrderbookSyncing bool
	EnableTradeSyncing     bool
	SyncWorkersCount       int
	SyncContinuously       bool
	SyncTimeoutREST        time.Duration
	SyncTimeoutWebsocket   time.Duration
}

// ExchangeTuningSettings defines settings related to an exchange
type ExchangeTuningSettings struct {
	EnableExchangeHTTPRateLimiter       bool
	EnableExchangeHTTPDebugging         bool
	EnableExchangeVerbose               bool
	ExchangePurgeCredentials            bool
	EnableExchangeAutoPairUpdates       bool
	DisableExchangeAutoPairUpdates      bool
	EnableExchangeRESTSupport           bool
	EnableExchangeWebsocketSupport      bool
	MaxHTTPRequestJobsLimit             int
	TradeBufferProcessingInterval       time.Duration
	RequestMaxRetryAttempts             int
	AlertSystemPreAllocationCommsBuffer int // See exchanges/alert.go
	ExchangeShutdownTimeout             time.Duration
	HTTPTimeout                         time.Duration
	HTTPUserAgent                       string
	HTTPProxy                           string
	GlobalHTTPTimeout                   time.Duration
	GlobalHTTPUserAgent                 string
	GlobalHTTPProxy                     string
}

const (
	// MsgStatusOK message to display when status is "OK"
	MsgStatusOK string = "ok"
	// MsgStatusSuccess message to display when status is successful
	MsgStatusSuccess string = "success"
	// MsgStatusError message to display when failure occurs
	MsgStatusError string = "error"
	grpcName       string = "grpc"
	grpcProxyName  string = "grpc_proxy"
)

// newConfigMutex only locks and unlocks on engine creation functions
// as engine modifies global files, this protects the main bot creation
// functions from interfering with each other
var newEngineMutex sync.Mutex

// Engine contains configuration, portfolio manager, exchange & ticker data and is the
// overarching type across this code base.
type MarketMakerEngine struct {
	Config *config.Config
	// communicationManager    *engine.CommunicationManager
	currencyPairSyncer *syncManager
	ExchangeManager    *ExchangeManager
	// OrderManager            *OrderManager
	MarketMaker *MarketMaker
	// websocketRoutineManager *websocketRoutineManager
	Settings           Settings
	uptime             time.Time
	GRPCShutdownSignal chan struct{}
	ServicesWG         sync.WaitGroup
}

func New() (*MarketMakerEngine, error) {
	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()

	var t MarketMakerEngine
	t.Config = config.GetConfig()

	err := t.Config.LoadConfig("~/.gocryptotrader/config.json", false)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %+v", err)
	}
	return &t, nil
}

func NewFromSetting(settings *Settings, flagset map[string]bool) (*MarketMakerEngine, error) {
	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()
	if settings == nil {
		return nil, errors.New("setting is nil")
	}

	var t MarketMakerEngine
	var err error
	t.Config, err = loadConfigWithSettings(settings, flagset)
	if err != nil {
		return nil, err
	}

	if *t.Config.Logging.Enabled {
		if err := gctlog.SetupGlobalLogger("gct/backtester", false); err != nil {
			return nil, fmt.Errorf("failed to setup global logger: %+v", err)
		}
		if err := gctlog.SetupSubLoggers(t.Config.Logging.SubLoggers); err != nil {
			return nil, fmt.Errorf("failed to setup sub logger: %+v", err)
		}
		gctlog.Infoln(gctlog.Global, "Logger initiated")
	}

	t.Settings = *settings
	t.Settings.ConfigFile = settings.ConfigFile
	t.Settings.DataDir = t.Config.GetDataPath()
	t.Settings.CheckParamInteraction = settings.CheckParamInteraction

	err = utils.AdjustGoMaxProcs(settings.GoMaxProcs)
	if err != nil {
		return nil, fmt.Errorf("unable to adjust runtime GOMAXPROCS value. Err: %w", err)
	}

	t.ExchangeManager = NewExchangeManager()
	// if t.Settings.EnableCommsRelayer {
	// 	if c, err := engine.SetupCommunicationManager(&t.Config.Communications); err != nil {
	// 		return nil, fmt.Errorf("failed to setup communication manager. Err: %+v", err)
	// 	} else {
	// 		t.communicationManager = c
	// 		if err := t.communicationManager.Start(); err != nil {
	// 			return nil, fmt.Errorf("failed to start communication manager. Err: %+v", err)
	// 		}
	// 	}
	// }

	return &t, nil
}

// loadConfigWithSettings creates configuration based on the provided settings
func loadConfigWithSettings(settings *Settings, flagSet map[string]bool) (*config.Config, error) {
	filePath, err := config.GetAndMigrateDefaultPath(settings.ConfigFile)
	if err != nil {
		return nil, err
	}
	log.Printf("Loading config file %s..\n", filePath)

	conf := &config.Config{}
	err = conf.ReadConfigFromFile(filePath, settings.EnableDryRun)
	if err != nil {
		return nil, fmt.Errorf("%+v %s: %+v", config.ErrFailureOpeningConfig, filePath, err)
	}
	// Apply overrides from settings
	if flagSet["datadir"] {
		// warn if dryrun isn't enabled
		if !settings.EnableDryRun {
			log.Println("Command line argument '-datadir' induces dry run mode.")
		}
		settings.EnableDryRun = true
		conf.DataDirectory = settings.DataDir
	}

	return conf, conf.CheckConfig()
}

// FlagSet defines set flags from command line args for comparison methods
type FlagSet map[string]bool

// WithBool checks the supplied flag. If set it will override the config boolean
// value as a command line takes precedence. If not set will fall back to config
// options.
func (f FlagSet) WithBool(key string, flagValue *bool, configValue bool) {
	isSet := f[key]
	*flagValue = !isSet && configValue || isSet && *flagValue
}

// PrintLoadedSettings logs loaded settings.
func (s *Settings) PrintLoadedSettings() {
	if s == nil {
		return
	}
	gctlog.Debugln(gctlog.Global)
	gctlog.Debugf(gctlog.Global, "ENGINE SETTINGS")
	settings := reflect.ValueOf(*s)
	for x := 0; x < settings.NumField(); x++ {
		field := settings.Field(x)
		if field.Kind() != reflect.Struct {
			continue
		}

		fieldName := field.Type().Name()
		gctlog.Debugln(gctlog.Global, "- "+common.AddPaddingOnUpperCase(fieldName)+":")
		for y := 0; y < field.NumField(); y++ {
			indvSetting := field.Field(y)
			indvName := field.Type().Field(y).Name
			if indvSetting.Kind() == reflect.String && indvSetting.IsZero() {
				indvSetting = reflect.ValueOf("Undefined")
			}
			gctlog.Debugln(gctlog.Global, "\t", common.AddPaddingOnUpperCase(indvName)+":", indvSetting)
		}
	}
	gctlog.Debugln(gctlog.Global)
}

func (te *MarketMakerEngine) Start() error {
	if te == nil {
		return errors.New("trading instance is nil")
	}

	var err error
	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()
	defer func() {
		if r := recover(); r != nil {
			gctlog.Errorf(gctlog.Global, "recover from panic: %+v", r)
		}
	}()

	if te.Settings.EnableDispatcher {
		if err := dispatch.Start(te.Settings.DispatchMaxWorkerAmount, te.Settings.DispatchJobsLimit); err != nil {
			gctlog.Errorf(gctlog.DispatchMgr, "error when start dispatch worker: %+v", err)
		}
	}

	te.uptime = time.Now()
	gctlog.Debugf(gctlog.Global, "fixengine '%s' started.\n", te.Config.Name)
	gctlog.Debugf(gctlog.Global, "Using data dir: %s\n", te.Settings.DataDir)
	if *te.Config.Logging.Enabled && strings.Contains(te.Config.Logging.Output, "file") {
		gctlog.Debugf(gctlog.Global,
			"Using log file: %s\n",
			filepath.Join(gctlog.GetLogPath(),
				te.Config.Logging.LoggerFileConfig.FileName),
		)
	}
	gctlog.Debugf(gctlog.Global,
		"Using %d out of %d logical processors for runtime performance\n",
		runtime.GOMAXPROCS(-1), te.Settings.GoMaxProcs)

	enabledExchanges := te.Config.CountEnabledExchanges()
	if te.Settings.EnableAllExchanges {
		enabledExchanges = len(te.Config.Exchanges)
	}

	gctlog.Debugln(gctlog.Global, "EXCHANGE COVERAGE")
	gctlog.Debugf(gctlog.Global, "\t Available Exchanges: %d. Enabled Exchanges: %d.\n",
		len(te.Config.Exchanges), enabledExchanges)

	if te.Settings.ExchangePurgeCredentials {
		gctlog.Debugln(gctlog.Global, "Purging exchange API credentials.")
		te.Config.PurgeExchangeAPICredentials()
	}

	gctlog.Debugln(gctlog.Global, "Setting up exchanges..")
	err = te.SetupExchanges()
	if err != nil {
		return err
	}

	if te.Settings.EnableExchangeSyncManager {
		exchangeSyncCfg := &SyncManagerConfig{
			SynchronizeTicker:       te.Settings.ExchangeSyncerSettings.EnableTickerSyncing,
			SynchronizeOrderbook:    te.Settings.ExchangeSyncerSettings.EnableOrderbookSyncing,
			SynchronizeTrades:       te.Settings.ExchangeSyncerSettings.EnableTradeSyncing,
			SynchronizeContinuously: te.Settings.ExchangeSyncerSettings.SyncContinuously,
			TimeoutREST:             te.Settings.ExchangeSyncerSettings.SyncTimeoutREST,
			TimeoutWebsocket:        te.Settings.ExchangeSyncerSettings.SyncTimeoutWebsocket,
			NumWorkers:              te.Settings.ExchangeSyncerSettings.SyncWorkersCount,
			Verbose:                 te.Settings.Verbose,
			FiatDisplayCurrency:     te.Config.Currency.FiatDisplayCurrency,
			PairFormatDisplay:       te.Config.Currency.CurrencyPairFormat,
		}

		te.currencyPairSyncer, err = setupSyncManager(
			exchangeSyncCfg,
			te.ExchangeManager,
			&te.Config.RemoteControl,
			true)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "Unable to initialise exchange currency pair syncer. Err: %s", err)
		} else {
			go func() {
				err = te.currencyPairSyncer.Start()
				if err != nil {
					gctlog.Errorf(gctlog.Global, "failed to start exchange currency pair manager. Err: %s", err)
				}
			}()
		}
	}

	// te.websocketRoutineManager, err = setupWebsocketRoutineManager(te.ExchangeManager, nil, te.currencyPairSyncer, &te.Config.Currency, te.Settings.Verbose)
	// if err != nil {
	// 	gctlog.Errorf(gctlog.Global, "Unable to initialise websocket routine manager. Err: %s", err)
	// } else {
	// 	err = te.websocketRoutineManager.Start()
	// 	if err != nil {
	// 		gctlog.Errorf(gctlog.Global, "failed to start websocket routine manager. Err: %s", err)
	// 	}
	// }
	// orderManager, err := SetupOrderManager(te.ExchangeManager, te.communicationManager, &te.ServicesWG, &te.Config.OrderManager)
	// if err != nil {
	// 	gctlog.Errorf(gctlog.Global, "Unable to initialise order manager. Err: %s", err)
	// }
	// if err := orderManager.Start(); err != nil {
	// 	gctlog.Errorf(gctlog.Global, "Unable to start order manager. Err: %s", err)
	// }
	// te.OrderManager = orderManager

	marketMaker, err := NewMarketMaker(te.ExchangeManager)
	if err != nil {
		gctlog.Errorf(gctlog.Global, "Unable to initiate market maker: %+v", err)
	}
	if err := marketMaker.Start(); err != nil {
		gctlog.Errorf(gctlog.Global, "Unable to start market maker: %+v", err)
	}
	te.MarketMaker = marketMaker
	return nil
}

// Stop correctly shuts down engine saving configuration files
func (tradeEngine *MarketMakerEngine) Stop() {
	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()

	gctlog.Debugln(gctlog.Global, "Engine shutting down..")
	if dispatch.IsRunning() {
		if err := dispatch.Stop(); err != nil {
			gctlog.Errorf(gctlog.DispatchMgr, "Dispatch system unable to stop. Error: %v", err)
		}
	}
	tradeEngine.MarketMaker.Stop()
	// if tradeEngine.websocketRoutineManager.IsRunning() {
	// 	if err := tradeEngine.websocketRoutineManager.Stop(); err != nil {
	// 		gctlog.Errorf(gctlog.Global, "websocket routine manager unable to stop. Error: %v", err)
	// 	}
	// }

	// if err := tradeEngine.OrderManager.Stop(); err != nil {
	// 	gctlog.Errorf(gctlog.Global, "Order manager unable to stop. Error: %v", err)
	// }

	// if err := tradeEngine.communicationManager.Stop(); err != nil {
	// 	gctlog.Errorf(gctlog.Global, "Communication manager unable to stop. Error: %v", err)
	// }

	err := tradeEngine.ExchangeManager.Shutdown(tradeEngine.Settings.ExchangeShutdownTimeout)
	if err != nil {
		gctlog.Errorf(gctlog.Global, "Exchange manager unable to stop. Error: %v", err)
	}

	// Wait for services to gracefully shutdown
	tradeEngine.ServicesWG.Wait()
	gctlog.Infoln(gctlog.Global, "Exiting.")
	if err := gctlog.CloseLogger(); err != nil {
		log.Printf("Failed to close logger. Error: %v\n", err)
	}
}

// GetExchangeByName returns an exchange given an exchange name
func (tradeEngine *MarketMakerEngine) GetExchangeByName(exchName string) (exchange.IBotExchange, error) {
	return tradeEngine.ExchangeManager.GetExchangeByName(exchName)
}

// UnloadExchange unloads an exchange by name
func (tradeEngine *MarketMakerEngine) UnloadExchange(exchName string) error {
	exchCfg, err := tradeEngine.Config.GetExchangeConfig(exchName)
	if err != nil {
		return err
	}

	err = tradeEngine.ExchangeManager.RemoveExchange(exchName)
	if err != nil {
		return err
	}

	exchCfg.Enabled = false
	return nil
}

// GetExchanges retrieves the loaded exchanges
func (tradeEngine *MarketMakerEngine) GetExchanges() []exchange.IBotExchange {
	exch, err := tradeEngine.ExchangeManager.GetExchanges()
	if err != nil {
		gctlog.Warnf(gctlog.ExchangeSys, "Cannot get exchanges: %v", err)
		return []exchange.IBotExchange{}
	}
	return exch
}

// LoadExchange loads an exchange by name. Optional wait group can be added for
// external synchronization.
func (tradeEngine *MarketMakerEngine) LoadExchange(name string, wg *sync.WaitGroup) error {
	exch, err := tradeEngine.ExchangeManager.NewExchangeByName(name)
	if err != nil {
		return err
	}
	if exch.GetBase() == nil {
		return ErrExchangeFailedToLoad
	}

	var localWG sync.WaitGroup
	localWG.Add(1)
	go func() {
		exch.SetDefaults()
		localWG.Done()
	}()
	exchCfg, err := tradeEngine.Config.GetExchangeConfig(name)
	if err != nil {
		return err
	}

	if tradeEngine.Settings.EnableAllPairs &&
		exchCfg.CurrencyPairs != nil {
		assets := exchCfg.CurrencyPairs.GetAssetTypes(false)
		for x := range assets {
			var pairs currency.Pairs
			pairs, err = exchCfg.CurrencyPairs.GetPairs(assets[x], false)
			if err != nil {
				return err
			}
			err = exchCfg.CurrencyPairs.StorePairs(assets[x], pairs, true)
			if err != nil {
				return err
			}
		}
	}

	if tradeEngine.Settings.EnableExchangeVerbose {
		exchCfg.Verbose = true
	}
	if exchCfg.Features != nil {
		if tradeEngine.Settings.EnableExchangeWebsocketSupport &&
			exchCfg.Features.Supports.Websocket {
			exchCfg.Features.Enabled.Websocket = true
		}
		if tradeEngine.Settings.EnableExchangeAutoPairUpdates &&
			exchCfg.Features.Supports.RESTCapabilities.AutoPairUpdates {
			exchCfg.Features.Enabled.AutoPairUpdates = true
		}
		if tradeEngine.Settings.DisableExchangeAutoPairUpdates {
			if exchCfg.Features.Supports.RESTCapabilities.AutoPairUpdates {
				exchCfg.Features.Enabled.AutoPairUpdates = false
			}
		}
	}
	if tradeEngine.Settings.HTTPUserAgent != "" {
		exchCfg.HTTPUserAgent = tradeEngine.Settings.HTTPUserAgent
	}
	if tradeEngine.Settings.HTTPProxy != "" {
		exchCfg.ProxyAddress = tradeEngine.Settings.HTTPProxy
	}
	if tradeEngine.Settings.HTTPTimeout != exchange.DefaultHTTPTimeout {
		exchCfg.HTTPTimeout = tradeEngine.Settings.HTTPTimeout
	}
	if tradeEngine.Settings.EnableExchangeHTTPDebugging {
		exchCfg.HTTPDebugging = tradeEngine.Settings.EnableExchangeHTTPDebugging
	}

	localWG.Wait()
	if !tradeEngine.Settings.EnableExchangeHTTPRateLimiter {
		gctlog.Warnf(gctlog.ExchangeSys,
			"Loaded exchange %s rate limiting has been turned off.\n",
			exch.GetName(),
		)
		err = exch.DisableRateLimiter()
		if err != nil {
			gctlog.Errorf(gctlog.ExchangeSys,
				"Loaded exchange %s rate limiting cannot be turned off: %s.\n",
				exch.GetName(),
				err,
			)
		}
	}

	// NOTE: This will standardize name to default and apply it to the config.
	exchCfg.Name = exch.GetName()

	exchCfg.Enabled = true
	err = exch.Setup(exchCfg)
	if err != nil {
		gctlog.Errorf(gctlog.ExchangeSys, "Exchange setup error: %+v", err)
		exchCfg.Enabled = false
		return err
	}

	err = tradeEngine.ExchangeManager.Add(exch)
	if err != nil {
		return err
	}

	base := exch.GetBase()
	if base.API.AuthenticatedSupport ||
		base.API.AuthenticatedWebsocketSupport {
		assetTypes := base.GetAssetTypes(false)
		var useAsset asset.Item
		for a := range assetTypes {
			err = base.CurrencyPairs.IsAssetEnabled(assetTypes[a])
			if err != nil {
				continue
			}
			useAsset = assetTypes[a]
			break
		}
		err = exch.ValidateAPICredentials(context.TODO(), useAsset)
		if err != nil {
			gctlog.Warnf(gctlog.ExchangeSys,
				"%s: Cannot validate credentials, authenticated support has been disabled, Error: %s\n",
				base.Name,
				err)
			base.API.AuthenticatedSupport = false
			base.API.AuthenticatedWebsocketSupport = false
			exchCfg.API.AuthenticatedSupport = false
			exchCfg.API.AuthenticatedWebsocketSupport = false
		}
	}

	return exchange.Bootstrap(context.TODO(), exch)
}

// SetupExchanges sets up the exchanges used by the fixengine
func (tradeEngine *MarketMakerEngine) SetupExchanges() error {
	var wg sync.WaitGroup
	configs := tradeEngine.Config.GetAllExchangeConfigs()

	for x := range configs {
		if !configs[x].Enabled && !tradeEngine.Settings.EnableAllExchanges {
			gctlog.Debugf(gctlog.ExchangeSys, "%s: Exchange support: Disabled\n", configs[x].Name)
			continue
		}
		wg.Add(1)
		go func(c config.Exchange) {
			defer wg.Done()
			err := tradeEngine.LoadExchange(c.Name, &wg)
			if err != nil {
				gctlog.Errorf(gctlog.ExchangeSys, "LoadExchange %s failed: %s\n", c.Name, err)
				return
			}
			gctlog.Debugf(gctlog.ExchangeSys,
				"%s: Exchange support: Enabled (Authenticated API support: %s - Verbose mode: %s).\n",
				c.Name,
				common.IsEnabled(c.API.AuthenticatedSupport),
				common.IsEnabled(c.Verbose),
			)
		}(configs[x])
	}
	wg.Wait()
	if len(tradeEngine.GetExchanges()) == 0 {
		return ErrNoExchangesLoaded
	}
	return nil
}
