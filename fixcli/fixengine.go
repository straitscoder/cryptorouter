package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/quickfixgo/fix42/newordersingle"
	"github.com/quickfixgo/quickfix"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
)

type fixApplication struct{}

func (c *fixApplication) OnCreate(sessionID quickfix.SessionID) {}

func (c *fixApplication) OnLogon(sessionID quickfix.SessionID) {}

func (c *fixApplication) OnLogout(sessionID quickfix.SessionID) {}

func (c *fixApplication) FromAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return nil
}

func (c *fixApplication) FromApp(msg *quickfix.Message, sessionID quickfix.SessionID) (reject quickfix.MessageRejectError) {
	return nil
}

func (c *fixApplication) ToAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) {}

func (c *fixApplication) ToApp(msg *quickfix.Message, sessionID quickfix.SessionID) error {
	return nil
}

func NewInitiator(settings *quickfix.Settings, storeFactory quickfix.MessageStoreFactory, logFactory *quickfix.LogFactory) (*quickfix.Initiator, error) {
	initiator, err := quickfix.NewInitiator(&fixApplication{}, storeFactory, settings, *logFactory)
	if err != nil {
		return nil, err
	}

	return initiator, nil
}

type FixEngine struct {
	initiator    *quickfix.Initiator
	settings     *quickfix.Settings
	logFactory   *quickfix.LogFactory
	storeFactory quickfix.MessageStoreFactory
}

func (fe *FixEngine) Start() error {
	var cfgFileName string
	fileName := "fixcli.cfg"
	execPath, _ := common.GetExecutablePath()
	cfgFileName = path.Join(execPath, fileName)

	if !file.Exists(cfgFileName) {
		cfgFileName = path.Join(common.GetDefaultDataDir(runtime.GOOS), fileName)
	}

	cfg, err := os.Open(cfgFileName)
	if err != nil {
		return fmt.Errorf("Error opening %v, %v\n", cfgFileName, err)
	}
	defer cfg.Close()
	stringData, readErr := io.ReadAll(cfg)
	if readErr != nil {
		return fmt.Errorf("Error reading cfg: %s,", readErr)
	}

	fe.settings, err = quickfix.ParseSettings(bytes.NewReader(stringData))
	if err != nil {
		return fmt.Errorf("Error reading setting cfg: %+v", err)
	}

	logFactory := quickfix.NewScreenLogFactory()
	fe.logFactory = &logFactory

	fe.storeFactory = quickfix.NewMemoryStoreFactory()

	initiator, err := NewInitiator(fe.settings, fe.storeFactory, fe.logFactory)
	if err != nil {
		return fmt.Errorf("error when initiate initiator : %+v", err)
	}
	fe.initiator = initiator
	if err := fe.initiator.Start(); err != nil {
		return fmt.Errorf("error when start initiator : %+v", err)
	}
	return nil
}

type NewOrderRequest struct {
	CliOrdID  string  `json:"cliOrdID"`
	Exchange  string  `json:"exchange"`
	Pair      string  `json:"pair"`
	Side      string  `json:"side"`
	Type      string  `json:"type"`
	Amount    float64 `json:"amount"`
	Price     float64 `json:"price"`
	Asset     string  `json:"asset"`
	HandleIns string  `json:"handleIns"`
}

func (fe *FixEngine) NewOrder(req *NewOrderRequest) (*quickfix.Message, error) {
	jsonOutput(req)
	var orderRequest newordersingle.NewOrderSingle
	orderRequest.SetClOrdID(req.CliOrdID)
	orderRequest.SetHandlInst(convertHandleInst(req.HandleIns))
	timestamp := time.Now().UTC()
	orderRequest.SetTransactTime(timestamp)
	orderRequest.SetOrdType(convertOrdType(req.Type))
	orderRequest.SetPrice(decimal.NewFromFloat(req.Price), 8)
	orderRequest.SetOrderQty(decimal.NewFromFloat(req.Amount), 8)
	orderRequest.SetSecurityExchange(strings.ToLower(req.Exchange))
	orderRequest.SetSecurityType(convertAsset(req.Asset))
	orderRequest.SetSide(convertSide(req.Side))
	orderRequest.SetSymbol(req.Pair)
	jsonOutput(orderRequest)
	orderMsg := orderRequest.ToMessage()

	if err := quickfix.Send(orderMsg); err != nil {
		return nil, err
	}

	return orderMsg, nil
}
