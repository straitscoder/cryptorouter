package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/fixcli/model"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
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
		if ordStatus == "4" || ordStatus == "3" {
			parsed := parseFIXMessage(msg)
			jsonOutput(parsed)
		}
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
	go fe.ExecutionReportRoutine()
	if err := fe.initiator.Start(); err != nil {
		return fmt.Errorf("error when start initiator : %+v", err)
	}
	return nil
}

func (fe *FixEngine) ExecutionReportRoutine() {
	fe.CheckExecutionReport()
	for {
		select {
		case <-time.After(time.Second * 1):
			go fe.CheckExecutionReport()
		}
	}
}

func (fe *FixEngine) CheckExecutionReport() {
	executionReport, err := model.GetExecutionReportQueue(context.Background())
	if err != nil {
		jsonOutput(err)
		return
	}
	if executionReport != nil {
		jsonOutput(executionReport)
		return
	}
	return
}

func (fe *FixEngine) NewOrder() error {
	clOrdId := generateClOrdID()
	symbol := Symbol()
	pair, err := currency.NewPairDelimiter(symbol, "-")
	if err != nil {
		return err
	}
	sideStr, _ := Side()
	ordTypeStr, _ := OrderType()
	// order := newordersingle.New(
	// 	field.NewClOrdID(clOrdId),
	// 	field.NewHandlInst(HandleIns()),
	// 	field.NewSymbol(symbol),
	// 	field.NewSide(sideFix),
	// 	field.NewTransactTime(time.Now().UTC()),
	// 	field.NewOrdType(ordTypeFix),
	// )
	assetStr, _ := AssetType()
	price := Price()
	amount := Amount()
	exchange := Exchange()
	// order.SetSecurityExchange(exchange)
	// order.SetSecurityType(securityType)
	// order.Set(field.NewPrice(price, 8))
	// order.Set(field.NewOrderQty(amount, 8))
	// orderMsg := order.ToMessage()
	// orderMsg.Header.Set(field.NewSenderCompID(fe.senderCompId))
	// orderMsg.Header.Set(field.NewTargetCompID(fe.targetCompId))
	// parsed := parseFIXMessage(orderMsg)
	// jsonOutput(parsed)
	if !Confirmation() {
		fmt.Println("Order canceled")
		return nil
	}
	rpcOrder := gctrpc.SubmitOrderRequest{
		ClientOrderId: clOrdId,
		Pair:          &gctrpc.CurrencyPair{Base: pair.Base.String(), Delimiter: pair.Delimiter, Quote: pair.Quote.String()},
		Exchange:      exchange,
		Side:          sideStr,
		OrderType:     ordTypeStr,
		Amount:        amount.InexactFloat64(),
		Price:         price.InexactFloat64(),
		AssetType:     assetStr,
	}
	if err := model.AddSubmitQueue(context.Background(), &rpcOrder); err != nil {
		return err
	}
	return nil
}

func (fe *FixEngine) CancelOrder() error {
	clOrdId := ClOrdID()
	orderId := getOrderId(clOrdId)
	sideStr, _ := Side()
	symbol := Symbol()
	pair, err := currency.NewPairDelimiter(symbol, "-")
	if err != nil {
		return err
	}
	// cancelReq := ordercancelrequest.New(
	// 	field.NewOrigClOrdID(clOrdId),
	// 	field.NewClOrdID(generateClOrdID()),
	// 	field.NewSymbol(symbol),
	// 	field.NewSide(sideFix),
	// 	field.NewTransactTime(time.Now().UTC()),
	// )
	assetStr, _ := AssetType()
	// if orderId != nil {
	// 	cancelReq.SetOrderID(*orderId)
	// } else if orderId == nil && assetType == enum.SecurityType_FUTURE {
	// 	cancelReq.SetOrderID(string(enum.SecurityType_FUTURE))
	// } else {
	if orderId == nil {
		fmt.Println("Order not found")
		return nil
	}
	exchange := Exchange()
	orderTypeStr, _ := OrderType()
	// cancelReq.SetSecurityExchange(exchange)
	// cancelReq.SetSecurityType(assetType)
	// cancelReqMsg := cancelReq.ToMessage()
	// cancelReqMsg.Header.Set(field.NewSenderCompID(fe.senderCompId))
	// cancelReqMsg.Header.Set(field.NewTargetCompID(fe.targetCompId))
	// parsed := parseFIXMessage(cancelReqMsg)
	// jsonOutput(parsed)
	if !Confirmation() {
		fmt.Println("Abort cancel order")
		return nil
	}
	cancelRpc := gctrpc.CancelOrderRequest{
		Exchange:      exchange,
		OrderId:       *orderId,
		ClientOrderId: clOrdId,
		Pair:          &gctrpc.CurrencyPair{Base: pair.Base.String(), Delimiter: pair.Delimiter, Quote: pair.Quote.String()},
		AssetType:     assetStr,
		Side:          sideStr,
		OrderType:     orderTypeStr,
	}
	if err := model.AddCancelQueue(context.Background(), &cancelRpc); err != nil {
		return err
	}
	deleteOrderId(clOrdId)
	return nil
}

func (fe *FixEngine) ModifyOrder() error {
	cliOrdId := ClOrdID()
	orderId := getOrderId(cliOrdId)
	exchange := Exchange()
	symbol := Symbol()
	pair, err := currency.NewPairDelimiter(symbol, "-")
	if err != nil {
		return err
	}
	sideStr, _ := Side()
	ordTypeStr, _ := OrderType()
	// modOrder := ordercancelreplacerequest.New(
	// 	field.NewOrigClOrdID(cliOrdId),
	// 	field.NewClOrdID(generateClOrdID()),
	// 	field.NewHandlInst(HandleIns()),
	// 	field.NewSymbol(symbol),
	// 	field.NewSide(sideFix),
	// 	field.NewTransactTime(time.Now().UTC()),
	// 	field.NewOrdType(orderType),
	// )

	assetDtr, _ := AssetType()
	price := Price()
	amount := Amount()
	// if orderId != nil {
	// 	modOrder.SetOrderID(*orderId)
	// } else if orderId == nil && assetType == enum.SecurityType_FUTURE {
	// 	modOrder.SetOrderID(string(enum.SecurityType_FUTURE))
	// } else {
	if orderId == nil {
		fmt.Println("Order not found")
		return nil
	}
	// modOrder.SetSecurityExchange(Exchange())
	// modOrder.SetSecurityType(assetType)
	// modOrder.Set(field.NewPrice(price, 8))
	// modOrder.Set(field.NewOrderQty(amount, 8))
	// modOrderMsg := modOrder.ToMessage()
	// modOrderMsg.Header.Set(field.NewSenderCompID(fe.senderCompId))
	// modOrderMsg.Header.Set(field.NewTargetCompID(fe.targetCompId))
	// parsed := parseFIXMessage(modOrderMsg)
	// jsonOutput(parsed)
	if !Confirmation() {
		fmt.Println("Abort modify order")
		return nil
	}
	modRpc := gctrpc.ModifyOrderRequest{
		Exchange:      exchange,
		OrderId:       *orderId,
		Pair:          &gctrpc.CurrencyPair{Base: pair.Base.String(), Delimiter: pair.Delimiter, Quote: pair.Quote.String()},
		Asset:         assetDtr,
		Amount:        amount.InexactFloat64(),
		Price:         price.InexactFloat64(),
		ClientOrderId: cliOrdId,
		Side:          sideStr,
		OrderType:     ordTypeStr,
	}
	if err := model.AddModifyQueue(context.Background(), &modRpc); err != nil {
		return err
	}
	return nil
}
