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
	EnableFIXEngine       bool

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
type FixEngine struct {
	Config *config.Config
	// apiServer               *apiServerManager
	// connectionManager       *connectionManager
	currencyPairSyncer      *syncManager
	ExchangeManager         *ExchangeManager
	websocketRoutineManager *websocketRoutineManager
	Settings                Settings
	uptime                  time.Time
	GRPCShutdownSignal      chan struct{}
	ServicesWG              sync.WaitGroup
	fixgateway              *Application
}

// New starts a new engine
func New() (*FixEngine, error) {
	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()
	var b FixEngine
	b.Config = config.GetConfig()

	err := b.Config.LoadConfig("~/.gocryptotrader/config.json", false)
	if err != nil {
		return nil, fmt.Errorf("failed to load config. Err: %s", err)
	}

	return &b, nil
}

// NewFromSettings starts a new engine based on supplied settings
func NewFromSettings(settings *Settings, flagSet map[string]bool) (*FixEngine, error) {
	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()
	if settings == nil {
		return nil, errors.New("engine: settings is nil")
	}

	var b FixEngine
	var err error

	b.Config, err = loadConfigWithSettings(settings, flagSet)
	if err != nil {
		return nil, fmt.Errorf("failed to load config. Err: %w", err)
	}

	if *b.Config.Logging.Enabled {
		err = gctlog.SetupGlobalLogger("gct/backtester", false)
		if err != nil {
			return nil, fmt.Errorf("failed to setup global logger. %w", err)
		}
		err = gctlog.SetupSubLoggers(b.Config.Logging.SubLoggers)
		if err != nil {
			return nil, fmt.Errorf("failed to setup sub loggers. %w", err)
		}
		gctlog.Infoln(gctlog.Global, "Logger initialised.")
	}

	b.Settings = *settings
	b.Settings.ConfigFile = settings.ConfigFile
	b.Settings.DataDir = b.Config.GetDataPath()
	b.Settings.CheckParamInteraction = settings.CheckParamInteraction

	err = utils.AdjustGoMaxProcs(settings.GoMaxProcs)
	if err != nil {
		return nil, fmt.Errorf("unable to adjust runtime GOMAXPROCS value. Err: %w", err)
	}

	b.ExchangeManager = NewExchangeManager()

	return &b, nil
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
		return nil, fmt.Errorf(config.ErrFailureOpeningConfig, filePath, err)
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

// Start starts the engine
func (fixengine *FixEngine) Start() error {
	if fixengine == nil {
		return errors.New("engine instance is nil")
	}
	var err error
	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()

	if fixengine.Settings.EnableDispatcher {
		if err = dispatch.Start(fixengine.Settings.DispatchMaxWorkerAmount, fixengine.Settings.DispatchJobsLimit); err != nil {
			gctlog.Errorf(gctlog.DispatchMgr, "Dispatcher unable to start: %v", err)
		}
	}

	fixengine.uptime = time.Now()
	gctlog.Debugf(gctlog.Global, "fixengine '%s' started.\n", fixengine.Config.Name)
	gctlog.Debugf(gctlog.Global, "Using data dir: %s\n", fixengine.Settings.DataDir)
	if *fixengine.Config.Logging.Enabled && strings.Contains(fixengine.Config.Logging.Output, "file") {
		gctlog.Debugf(gctlog.Global,
			"Using log file: %s\n",
			filepath.Join(gctlog.GetLogPath(),
				fixengine.Config.Logging.LoggerFileConfig.FileName),
		)
	}
	gctlog.Debugf(gctlog.Global,
		"Using %d out of %d logical processors for runtime performance\n",
		runtime.GOMAXPROCS(-1), runtime.NumCPU())

	enabledExchanges := fixengine.Config.CountEnabledExchanges()
	if fixengine.Settings.EnableAllExchanges {
		enabledExchanges = len(fixengine.Config.Exchanges)
	}

	gctlog.Debugln(gctlog.Global, "EXCHANGE COVERAGE")
	gctlog.Debugf(gctlog.Global, "\t Available Exchanges: %d. Enabled Exchanges: %d.\n",
		len(fixengine.Config.Exchanges), enabledExchanges)

	if fixengine.Settings.ExchangePurgeCredentials {
		gctlog.Debugln(gctlog.Global, "Purging exchange API credentials.")
		fixengine.Config.PurgeExchangeAPICredentials()
	}

	gctlog.Debugln(gctlog.Global, "Setting up exchanges..")
	err = fixengine.SetupExchanges()
	if err != nil {
		return err
	}

	if fixengine.Settings.EnableExchangeSyncManager {
		exchangeSyncCfg := &SyncManagerConfig{
			SynchronizeTicker:       fixengine.Settings.ExchangeSyncerSettings.EnableTickerSyncing,
			SynchronizeOrderbook:    fixengine.Settings.ExchangeSyncerSettings.EnableOrderbookSyncing,
			SynchronizeTrades:       fixengine.Settings.ExchangeSyncerSettings.EnableTradeSyncing,
			SynchronizeContinuously: fixengine.Settings.ExchangeSyncerSettings.SyncContinuously,
			TimeoutREST:             fixengine.Settings.ExchangeSyncerSettings.SyncTimeoutREST,
			TimeoutWebsocket:        fixengine.Settings.ExchangeSyncerSettings.SyncTimeoutWebsocket,
			NumWorkers:              fixengine.Settings.ExchangeSyncerSettings.SyncWorkersCount,
			Verbose:                 fixengine.Settings.Verbose,
			FiatDisplayCurrency:     fixengine.Config.Currency.FiatDisplayCurrency,
			PairFormatDisplay:       fixengine.Config.Currency.CurrencyPairFormat,
		}

		fixengine.currencyPairSyncer, err = setupSyncManager(
			exchangeSyncCfg,
			fixengine.ExchangeManager,
			&fixengine.Config.RemoteControl,
			true)
		if err != nil {
			gctlog.Errorf(gctlog.Global, "Unable to initialise exchange currency pair syncer. Err: %s", err)
		} else {
			go func() {
				err = fixengine.currencyPairSyncer.Start()
				if err != nil {
					gctlog.Errorf(gctlog.Global, "failed to start exchange currency pair manager. Err: %s", err)
				}
			}()
		}
	}

	fixengine.websocketRoutineManager, err = setupWebsocketRoutineManager(fixengine.ExchangeManager, nil, fixengine.currencyPairSyncer, &fixengine.Config.Currency, fixengine.Settings.Verbose)
	if err != nil {
		gctlog.Errorf(gctlog.Global, "Unable to initialise websocket routine manager. Err: %s", err)
	} else {
		err = fixengine.websocketRoutineManager.Start()
		if err != nil {
			gctlog.Errorf(gctlog.Global, "failed to start websocket routine manager. Err: %s", err)
		}
	}

	fixengine.fixgateway = NewFixGateway(fixengine.websocketRoutineManager, fixengine.ExchangeManager)
	fixengine.fixgateway.Start()
	return nil
}

// Stop correctly shuts down engine saving configuration files
func (fixengine *FixEngine) Stop() {
	newEngineMutex.Lock()
	defer newEngineMutex.Unlock()

	gctlog.Debugln(gctlog.Global, "Engine shutting down..")
	if fixengine.fixgateway != nil {
		fixengine.fixgateway.Stop()
	}

	if dispatch.IsRunning() {
		if err := dispatch.Stop(); err != nil {
			gctlog.Errorf(gctlog.DispatchMgr, "Dispatch system unable to stop. Error: %v", err)
		}
	}
	if fixengine.websocketRoutineManager.IsRunning() {
		if err := fixengine.websocketRoutineManager.Stop(); err != nil {
			gctlog.Errorf(gctlog.Global, "websocket routine manager unable to stop. Error: %v", err)
		}
	}

	err := fixengine.ExchangeManager.Shutdown(fixengine.Settings.ExchangeShutdownTimeout)
	if err != nil {
		gctlog.Errorf(gctlog.Global, "Exchange manager unable to stop. Error: %v", err)
	}

	// Wait for services to gracefully shutdown
	fixengine.ServicesWG.Wait()
	gctlog.Infoln(gctlog.Global, "Exiting.")
	if err := gctlog.CloseLogger(); err != nil {
		log.Printf("Failed to close logger. Error: %v\n", err)
	}
}

// GetExchangeByName returns an exchange given an exchange name
func (fixengine *FixEngine) GetExchangeByName(exchName string) (exchange.IBotExchange, error) {
	return fixengine.ExchangeManager.GetExchangeByName(exchName)
}

// UnloadExchange unloads an exchange by name
func (fixengine *FixEngine) UnloadExchange(exchName string) error {
	exchCfg, err := fixengine.Config.GetExchangeConfig(exchName)
	if err != nil {
		return err
	}

	err = fixengine.ExchangeManager.RemoveExchange(exchName)
	if err != nil {
		return err
	}

	exchCfg.Enabled = false
	return nil
}

// GetExchanges retrieves the loaded exchanges
func (fixengine *FixEngine) GetExchanges() []exchange.IBotExchange {
	exch, err := fixengine.ExchangeManager.GetExchanges()
	if err != nil {
		gctlog.Warnf(gctlog.ExchangeSys, "Cannot get exchanges: %v", err)
		return []exchange.IBotExchange{}
	}
	return exch
}

// LoadExchange loads an exchange by name. Optional wait group can be added for
// external synchronization.
func (fixengine *FixEngine) LoadExchange(name string, wg *sync.WaitGroup) error {
	exch, err := fixengine.ExchangeManager.NewExchangeByName(name)
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
	exchCfg, err := fixengine.Config.GetExchangeConfig(name)
	if err != nil {
		return err
	}

	if fixengine.Settings.EnableAllPairs &&
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

	if fixengine.Settings.EnableExchangeVerbose {
		exchCfg.Verbose = true
	}
	if exchCfg.Features != nil {
		if fixengine.Settings.EnableExchangeWebsocketSupport &&
			exchCfg.Features.Supports.Websocket {
			exchCfg.Features.Enabled.Websocket = true
		}
		if fixengine.Settings.EnableExchangeAutoPairUpdates &&
			exchCfg.Features.Supports.RESTCapabilities.AutoPairUpdates {
			exchCfg.Features.Enabled.AutoPairUpdates = true
		}
		if fixengine.Settings.DisableExchangeAutoPairUpdates {
			if exchCfg.Features.Supports.RESTCapabilities.AutoPairUpdates {
				exchCfg.Features.Enabled.AutoPairUpdates = false
			}
		}
	}
	if fixengine.Settings.HTTPUserAgent != "" {
		exchCfg.HTTPUserAgent = fixengine.Settings.HTTPUserAgent
	}
	if fixengine.Settings.HTTPProxy != "" {
		exchCfg.ProxyAddress = fixengine.Settings.HTTPProxy
	}
	if fixengine.Settings.HTTPTimeout != exchange.DefaultHTTPTimeout {
		exchCfg.HTTPTimeout = fixengine.Settings.HTTPTimeout
	}
	if fixengine.Settings.EnableExchangeHTTPDebugging {
		exchCfg.HTTPDebugging = fixengine.Settings.EnableExchangeHTTPDebugging
	}

	localWG.Wait()
	if !fixengine.Settings.EnableExchangeHTTPRateLimiter {
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
		exchCfg.Enabled = false
		return err
	}

	err = fixengine.ExchangeManager.Add(exch)
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
func (fixengine *FixEngine) SetupExchanges() error {
	var wg sync.WaitGroup
	configs := fixengine.Config.GetAllExchangeConfigs()

	for x := range configs {
		if !configs[x].Enabled && !fixengine.Settings.EnableAllExchanges {
			gctlog.Debugf(gctlog.ExchangeSys, "%s: Exchange support: Disabled\n", configs[x].Name)
			continue
		}
		wg.Add(1)
		go func(c config.Exchange) {
			defer wg.Done()
			err := fixengine.LoadExchange(c.Name, &wg)
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
	if len(fixengine.GetExchanges()) == 0 {
		return ErrNoExchangesLoaded
	}
	return nil
}

// WaitForInitialCurrencySync allows for a routine to wait for the initial sync
// of the currency pair syncer management system.
func (fixengine *FixEngine) WaitForInitialCurrencySync() error {
	return fixengine.currencyPairSyncer.WaitForInitialSync()
}
