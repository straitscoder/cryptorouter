package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/quickfixgo/field"
	"github.com/quickfixgo/fix42/marketdatarequest"
	"github.com/quickfixgo/fix42/newordersingle"
	"github.com/quickfixgo/fix42/ordercancelreplacerequest"
	"github.com/quickfixgo/fix42/ordercancelrequest"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"gopkg.in/ini.v1"
)

type fixApplication struct {
	*quickfix.MessageRouter
}

func (c *fixApplication) OnCreate(sessionID quickfix.SessionID) {}

func (c *fixApplication) OnLogon(sessionID quickfix.SessionID) {}

func (c *fixApplication) OnLogout(sessionID quickfix.SessionID) {}

func (c *fixApplication) FromAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return nil
}

func (c *fixApplication) FromApp(msg *quickfix.Message, sessionID quickfix.SessionID) (reject quickfix.MessageRejectError) {
	msgType, _ := msg.Header.GetString(tag.MsgType)
	switch msgType {
	case "8":
		clOrdID, _ := msg.Body.GetString(tag.ClOrdID)
		orderId, _ := msg.Body.GetString(tag.OrderID)
		ordStatus, _ := msg.Body.GetString(tag.OrdStatus)
		savedOrderId := getOrderId(clOrdID)
		if savedOrderId != nil {
			if ordStatus != "0" {
				parsed := parseFIXMessage(msg)
				jsonOutput(parsed)
			}
			return nil
		} else {
			saveOrderId(orderId, clOrdID)
			if ordStatus == "0" {
				parsed := parseFIXMessage(msg)
				jsonOutput(parsed)
				orderResponse := make(map[string]string)
				orderResponse["Client_Order_ID"] = clOrdID
				orderResponse["Order_ID"] = orderId
				jsonOutput(orderResponse)
			}
		}
	case "W":
		parsed := parseFIXMessage(msg)
		jsonOutput(parsed)
	}
	return nil
}

func (c *fixApplication) ToAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) {}

func (c *fixApplication) ToApp(msg *quickfix.Message, sessionID quickfix.SessionID) error {
	return nil
}

func (c *fixApplication) NewOrderSingle(msg *quickfix.Message, sessionID quickfix.SessionID) error {
	return quickfix.Send(msg)
}

func NewInitiator(settings *quickfix.Settings, storeFactory quickfix.MessageStoreFactory, logFactory *quickfix.LogFactory) (*quickfix.Initiator, error) {
	app := &fixApplication{MessageRouter: quickfix.NewMessageRouter()}
	initiator, err := quickfix.NewInitiator(app, storeFactory, settings, *logFactory)
	if err != nil {
		return nil, err
	}

	return initiator, nil
}

type FixEngine struct {
	senderCompId string
	targetCompId string
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
		return fmt.Errorf("error opening %v, %v", cfgFileName, err)
	}
	defer cfg.Close()
	stringData, readErr := io.ReadAll(cfg)
	if readErr != nil {
		return fmt.Errorf("error reading cfg: %s,", readErr)
	}

	config, err := ini.Load(cfgFileName)
	if err != nil {
		return fmt.Errorf("error reading cfg: %s,", err)
	}
	fe.senderCompId = config.Section("DEFAULT").Key("SenderCompID").String()
	fe.targetCompId = config.Section("SESSION").Key("TargetCompID").String()

	fe.settings, err = quickfix.ParseSettings(bytes.NewReader(stringData))
	if err != nil {
		return fmt.Errorf("error reading setting cfg: %+v", err)
	}

	logFactory, err := quickfix.NewFileLogFactory(fe.settings)
	if err != nil {
		return fmt.Errorf("unable to create logger: %s", err)
	}
	fe.logFactory = &logFactory

	fe.storeFactory = quickfix.NewMemoryStoreFactory()

	app := &fixApplication{MessageRouter: quickfix.NewMessageRouter()}
	initiator, err := quickfix.NewInitiator(app, fe.storeFactory, fe.settings, *fe.logFactory)
	if err != nil {
		return fmt.Errorf("error when initiate initiator : %+v", err)
	}
	fe.initiator = initiator
	if err := fe.initiator.Start(); err != nil {
		return fmt.Errorf("error when start initiator : %+v", err)
	}
	return nil
}

func (fe *FixEngine) NewOrder() error {
	clOrdId := generateClOrdID()
	order := newordersingle.New(
		field.NewClOrdID(clOrdId),
		field.NewHandlInst(HandleIns()),
		field.NewSymbol(Symbol()),
		field.NewSide(Side()),
		field.NewTransactTime(time.Now().UTC()),
		field.NewOrdType(OrderType()),
	)
	securityType := AssetType()
	order.SetSecurityExchange(Exchange())
	order.SetSecurityType(securityType)
	order.Set(field.NewPrice(Price(), 8))
	order.Set(field.NewOrderQty(Amount(), 8))
	orderMsg := order.ToMessage()
	orderMsg.Header.Set(field.NewSenderCompID(fe.senderCompId))
	orderMsg.Header.Set(field.NewTargetCompID(fe.targetCompId))
	parsed := parseFIXMessage(orderMsg)
	jsonOutput(parsed)
	if !Confirmation() {
		fmt.Println("Order canceled")
		return nil
	}
	return quickfix.Send(orderMsg)
}

func (fe *FixEngine) CancelOrder() error {
	clOrdId := ClOrdID()
	orderId := getOrderId(clOrdId)
	cancelReq := ordercancelrequest.New(
		field.NewOrigClOrdID(clOrdId),
		field.NewClOrdID(generateClOrdID()),
		field.NewSymbol(Symbol()),
		field.NewSide(Side()),
		field.NewTransactTime(time.Now().UTC()),
	)
	assetType := AssetType()
	if orderId != nil {
		cancelReq.SetOrderID(*orderId)
	} else {
		fmt.Println("Order not found")
		return nil
	}
	cancelReq.SetSecurityExchange(Exchange())
	cancelReq.SetSecurityType(assetType)
	cancelReqMsg := cancelReq.ToMessage()
	cancelReqMsg.Header.Set(field.NewSenderCompID(fe.senderCompId))
	cancelReqMsg.Header.Set(field.NewTargetCompID(fe.targetCompId))
	parsed := parseFIXMessage(cancelReqMsg)
	jsonOutput(parsed)
	if !Confirmation() {
		fmt.Println("Abort cancel order")
		return nil
	}
	deleteOrderId(clOrdId)
	return quickfix.Send(cancelReqMsg)
}

func (fe *FixEngine) ModifyOrder() error {
	cliOrdId := ClOrdID()
	orderId := getOrderId(cliOrdId)
	modOrder := ordercancelreplacerequest.New(
		field.NewOrigClOrdID(cliOrdId),
		field.NewClOrdID(generateClOrdID()),
		field.NewHandlInst(HandleIns()),
		field.NewSymbol(Symbol()),
		field.NewSide(Side()),
		field.NewTransactTime(time.Now().UTC()),
		field.NewOrdType(OrderType()),
	)

	assetType := AssetType()
	if orderId != nil {
		modOrder.SetOrderID(*orderId)
	} else {
		fmt.Println("Order not found")
		return nil
	}
	modOrder.SetSecurityExchange(Exchange())
	modOrder.SetSecurityType(assetType)
	modOrder.Set(field.NewPrice(Price(), 8))
	modOrder.Set(field.NewOrderQty(Amount(), 8))
	modOrderMsg := modOrder.ToMessage()
	modOrderMsg.Header.Set(field.NewSenderCompID(fe.senderCompId))
	modOrderMsg.Header.Set(field.NewTargetCompID(fe.targetCompId))
	parsed := parseFIXMessage(modOrderMsg)
	jsonOutput(parsed)
	if !Confirmation() {
		fmt.Println("Abort modify order")
		return nil
	}
	return quickfix.Send(modOrderMsg)
}

func (fe *FixEngine) MarketDataRequest() error {
	marketRequest := marketdatarequest.New(
		field.NewMDReqID(generateClOrdID()),
		field.NewSubscriptionRequestType(SubsReqType()),
		field.NewMarketDepth(MarketDepth()),
	)

	marketRequest.Set(field.NewMDUpdateType(MDUpdateType()))
	marketRequest.Set(field.NewNoMDEntryTypes(1))
	marketRequest.Set(field.NewMDEntryType(MDEntryType()))
	marketRequest.Set(field.NewNoRelatedSym(1))
	marketRequest.Set(field.NewSymbol(Symbol()))
	marketRequest.Set(field.NewSecurityExchange(Exchange()))
	marketRequest.Set(field.NewSecurityType(AssetType()))
	marketRequestMsg := marketRequest.ToMessage()
	marketRequestMsg.Header.Set(field.NewSenderCompID(fe.senderCompId))
	marketRequestMsg.Header.Set(field.NewTargetCompID(fe.targetCompId))
	parsed := parseFIXMessage(marketRequestMsg)
	jsonOutput(parsed)
	return quickfix.Send(marketRequestMsg)
}
