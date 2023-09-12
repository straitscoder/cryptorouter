package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	"github.com/quickfixgo/fix42/executionreport"
	"github.com/quickfixgo/fix42/marketdatarequest"
	"github.com/quickfixgo/fix42/marketdatasnapshotfullrefresh"
	"github.com/quickfixgo/fix42/newordersingle"
	"github.com/quickfixgo/fix42/ordercancelreplacerequest"
	"github.com/quickfixgo/fix42/ordercancelrequest"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/file"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
)

// Application implements the quickfix.Application interface
type Application struct {
	*quickfix.MessageRouter
	execID                int
	exchangeManager       *ExchangeManager
	pairFormater          *currency.PairFormat
	acceptor              *quickfix.Acceptor
	settings              *quickfix.Settings
	logFactory            *quickfix.LogFactory
	storeFactory          quickfix.MessageStoreFactory
	socketDispatcher      *websocketRoutineManager
	marketdataSubscribers map[string]quickfix.SessionID
	sessions              map[string]quickfix.SessionID
}

func NewFixGateway(eventDispacher *websocketRoutineManager, exchangeManager *ExchangeManager) *Application {
	app := &Application{MessageRouter: quickfix.NewMessageRouter()}
	app.AddRoute(newordersingle.Route(app.onNewOrderSingle))
	app.AddRoute(ordercancelrequest.Route(app.onOrderCancelRequest))
	app.AddRoute(marketdatarequest.Route(app.onMarketDataRequest))
	app.AddRoute(ordercancelreplacerequest.Route(app.onOrderCancelReplaceRequest))

	app.socketDispatcher = eventDispacher
	app.sessions = make(map[string]quickfix.SessionID)
	app.marketdataSubscribers = make(map[string]quickfix.SessionID)

	app.pairFormater = &currency.PairFormat{
		Uppercase: true,
		Delimiter: "-",
	}

	app.exchangeManager = exchangeManager
	app.execID = int(time.Now().Unix())
	return app
}

func (a *Application) Start() error {
	var cfgFileName string
	filename := "fixgw.cfg"
	execPath, _ := common.GetExecutablePath()
	cfgFileName = path.Join(execPath, filename)
	if !file.Exists(cfgFileName) {
		cfgFileName = path.Join(common.GetDefaultDataDir(runtime.GOOS), filename)
	}

	cfg, err := os.Open(cfgFileName)
	if err != nil {
		return fmt.Errorf("Error opening %v, %v\n", cfgFileName, err)
	}
	defer cfg.Close()
	stringData, readErr := ioutil.ReadAll(cfg)
	if readErr != nil {
		return fmt.Errorf("Error reading cfg: %s,", readErr)
	}

	a.settings, err = quickfix.ParseSettings(bytes.NewReader(stringData))
	if err != nil {
		return fmt.Errorf("Error reading cfg: %s,", err)
	}

	logFactory := quickfix.NewScreenLogFactory()
	//logFactory, err := quickfix.NewFileLogFactory(settings)
	if err != nil {
		return fmt.Errorf("Unable to create logger: %s\n", err)
	}
	a.logFactory = &logFactory

	a.storeFactory = quickfix.NewMemoryStoreFactory()
	//a.storeFactory = quickfix.NewFileStoreFactory(a.settings)
	a.acceptor, err = quickfix.NewAcceptor(a, a.storeFactory, a.settings, logFactory)
	if err != nil {
		return fmt.Errorf("Unable to create Acceptor: %s\n", err)
	}
	err = a.acceptor.Start()
	if err != nil {
		return fmt.Errorf("Unable to start Acceptor: %s\n", err)
	}

	a.socketDispatcher.registerWebsocketDataHandler(a.WebsocketDataHandler, false)
	return nil
}

func (a *Application) Stop() {
	a.acceptor.Stop()
}

// OnCreate implemented as part of Application interface
func (a *Application) OnCreate(sessionID quickfix.SessionID) {}

// OnLogon implemented as part of Application interface
func (a *Application) OnLogon(sessionID quickfix.SessionID) {
	a.sessions[sessionID.String()] = sessionID
}

// OnLogout implemented as part of Application interface
func (a *Application) OnLogout(sessionID quickfix.SessionID) {
	delete(a.marketdataSubscribers, sessionID.String())
	delete(a.sessions, sessionID.String())
}

// ToAdmin implemented as part of Application interface
func (a *Application) ToAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) {}

// ToApp implemented as part of Application interface
func (a *Application) ToApp(msg *quickfix.Message, sessionID quickfix.SessionID) error {
	return nil
}

// FromAdmin implemented as part of Application interface
func (a *Application) FromAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return nil
}

// FromApp implemented as part of Application interface, uses Router on incoming application messages
func (a *Application) FromApp(msg *quickfix.Message, sessionID quickfix.SessionID) (reject quickfix.MessageRejectError) {
	return a.Route(msg, sessionID)
}

func FromSide(side enum.Side) order.Side {
	switch side {
	case enum.Side_BUY:
		return order.Buy
	case enum.Side_SELL:
		return order.Sell
	default:
		return order.UnknownSide
	}
}

func FromOrdType(orderType enum.OrdType) order.Type {
	switch orderType {
	case enum.OrdType_LIMIT:
		return order.Limit
	case enum.OrdType_MARKET:
		return order.Market
	case enum.OrdType_STOP:
		return order.Stop
	case enum.OrdType_STOP_LIMIT:
		return order.StopLimit
	default:
		return order.UnknownType
	}
}

func FromSecurityType(secType enum.SecurityType) asset.Item {
	switch secType {
	case enum.SecurityType_FUTURE:
		return asset.Futures
	case enum.SecurityType_FX_SPOT:
		return asset.Spot
	case enum.SecurityType_NON_DELIVERABLE_FORWARD:
		return asset.Margin
	default:
		return asset.Empty
	}
}

func (a *Application) onNewOrderSingle(msg newordersingle.NewOrderSingle, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	clOrdID, err := msg.GetClOrdID()
	if err != nil {
		return err
	}

	symbol, err := msg.GetSymbol()
	if err != nil {
		return err
	}

	side, err := msg.GetSide()
	if err != nil {
		return err
	}

	ordType, err := msg.GetOrdType()
	if err != nil {
		return err
	}

	price, err := msg.GetPrice()
	if err != nil {
		return err
	}

	orderQty, err := msg.GetOrderQty()
	if err != nil {
		return err
	}

	exchange, err := msg.GetSecurityExchange()
	if err != nil {
		return err
	}

	securityType, err := msg.GetSecurityType()
	if err != nil {
		return err
	}

	pair, e := currency.NewPairFromString(symbol)
	if e != nil {
		return err
	}

	submission := &order.Submit{
		Pair:          pair,
		Side:          FromSide(side),
		Type:          FromOrdType(ordType),
		Amount:        orderQty.InexactFloat64(),
		Price:         price.InexactFloat64(),
		ClientOrderID: clOrdID,
		Exchange:      exchange,
		AssetType:     asset.Item(FromSecurityType(securityType)),
	}

	exch, e := a.exchangeManager.GetExchangeByName(submission.Exchange)
	if err != nil {
		a.RejectOrderRequest(submission, e.Error())
	}

	// Checks for exchange min max limits for order amounts before order
	// execution can occur
	e = exch.CheckOrderExecutionLimits(submission.AssetType,
		submission.Pair,
		submission.Price,
		submission.Amount,
		submission.Type)
	if e != nil {
		msg := fmt.Errorf("order manager: exchange %s unable to place order: %w",
			submission.Exchange,
			e)
		a.RejectOrderRequest(submission, msg.Error())
		return nil
	}

	// Determines if current trading activity is turned off by the exchange for
	// the currency pair
	e = exch.CanTradePair(submission.Pair, submission.AssetType)
	if e != nil {
		msg := fmt.Errorf("order manager: exchange %s cannot trade pair %s %s: %w",
			submission.Exchange,
			submission.Pair,
			submission.AssetType,
			e)
		a.RejectOrderRequest(submission, msg.Error())
	}

	_, e = exch.SubmitOrder(context.TODO(), submission)
	if e != nil {
		a.RejectOrderRequest(submission, e.Error())
	}
	return nil
}

func (a *Application) onOrderCancelRequest(msg ordercancelrequest.OrderCancelRequest, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	orderID, err := msg.GetOrderID()
	if err != nil {
		return err
	}

	side, err := msg.GetSide()
	if err != nil {
		return err
	}

	symbol, err := msg.GetSymbol()
	if err != nil {
		return err
	}

	clOrdID, err := msg.GetClOrdID()
	if err != nil {
		return err
	}

	exchange, err := msg.GetSecurityExchange()
	if err != nil {
		return err
	}

	securityType, err := msg.GetSecurityType()
	if err != nil {
		return err
	}

	pair, e := currency.NewPairFromString(symbol)
	if e != nil {
		return err
	}

	request := &order.Cancel{
		Exchange:      exchange,
		OrderID:       orderID,
		Side:          FromSide(side),
		Pair:          pair,
		AssetType:     FromSecurityType(securityType),
		ClientOrderID: clOrdID,
	}

	exch, e := a.exchangeManager.GetExchangeByName(request.Exchange)
	if e != nil {
		return quickfix.ValueIsIncorrect(tag.SecurityExchange)
	}

	exch.CancelOrder(context.TODO(), request)
	return nil
}

func (a *Application) onOrderCancelReplaceRequest(msg ordercancelreplacerequest.OrderCancelReplaceRequest, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	orderID, err := msg.GetOrderID()
	if err != nil {
		return err
	}

	side, err := msg.GetSide()
	if err != nil {
		return err
	}

	symbol, err := msg.GetSymbol()
	if err != nil {
		return err
	}

	clOrdID, err := msg.GetClOrdID()
	if err != nil {
		return err
	}

	exchange, err := msg.GetSecurityExchange()
	if err != nil {
		return err
	}

	securityType, err := msg.GetSecurityType()
	if err != nil {
		return err
	}

	pair, e := currency.NewPairFromString(symbol)
	if e != nil {
		return err
	}

	price, err := msg.GetPrice()
	if err != nil {
		return err
	}

	orderQty, err := msg.GetOrderQty()
	if err != nil {
		return err
	}

	request := &order.Modify{
		Exchange:      exchange,
		OrderID:       orderID,
		Side:          FromSide(side),
		Pair:          pair,
		AssetType:     FromSecurityType(securityType),
		ClientOrderID: clOrdID,
		Price:         price.InexactFloat64(),
		Amount:        orderQty.InexactFloat64(),
	}

	exch, e := a.exchangeManager.GetExchangeByName(request.Exchange)
	if e != nil {
		return quickfix.ValueIsIncorrect(tag.SecurityExchange)
	}

	exch.ModifyOrder(context.TODO(), request)
	return nil
}

func (a *Application) onMarketDataRequest(msg marketdatarequest.MarketDataRequest, sessionID quickfix.SessionID) (err quickfix.MessageRejectError) {
	a.marketdataSubscribers[sessionID.String()] = sessionID
	return
}

func ToOrdStatus(status order.Status) enum.OrdStatus {
	switch status {
	case order.Active, order.New:
		return enum.OrdStatus_NEW
	case order.Filled:
		return enum.OrdStatus_FILLED
	case order.PartiallyFilled:
		return enum.OrdStatus_PARTIALLY_FILLED
	case order.PartiallyCancelled, order.Cancelled:
		return enum.OrdStatus_CANCELED
	default:
		return enum.OrdStatus_REJECTED
	}
}

func (a *Application) WebsocketDataHandler(exchName string, data interface{}) error {
	if a == nil {
		return nil
	}

	switch d := data.(type) {
	case string:
		//log.Infoln(log.WebsocketMgr, d)
	case error:
		return fmt.Errorf("exchange %s websocket error - %s", exchName, data)
	case *ticker.Price:
		a.BroadcastMarketData(d)
	case *orderbook.Depth:
		a.BroadcastDepth(d)
	case *order.Detail:
		a.UpdateOrder(d, ToOrdStatus(d.Status))
	case order.ClassificationError:
		return fmt.Errorf("%w %s", d.Err, d.Error())
	case account.Change:
	case []trade.Data:
	case []fill.Data:
	default:
	}
	return nil
}

func ToSecurityType(assetType asset.Item) enum.SecurityType {
	switch assetType {
	case asset.Futures:
		return enum.SecurityType_FUTURE
	case asset.Margin:
		return enum.SecurityType_NON_DELIVERABLE_FORWARD
	case asset.Spot:
		return enum.SecurityType_FX_SPOT
	default:
		return enum.SecurityType_FX_FORWARD
	}
}

func (a *Application) BroadcastMarketData(price *ticker.Price) {
	symbol := a.pairFormater.Format(price.Pair)
	msg := marketdatasnapshotfullrefresh.New(field.NewSymbol(symbol))
	msg.SetSymbol(symbol)
	msg.SetSecurityExchange(price.ExchangeName)
	msg.SetSecurityType(ToSecurityType(price.AssetType))

	msg.SetTotalVolumeTraded(decimal.NewFromFloat(price.QuoteVolume), 2)

	mdEntries := marketdatasnapshotfullrefresh.NewNoMDEntriesRepeatingGroup()
	mdEntry := mdEntries.Add()
	mdEntry.SetMDEntryType(enum.MDEntryType_TRADING_SESSION_HIGH_PRICE)
	mdEntry.SetMDEntryPx(decimal.NewFromFloat(price.High), 8)

	mdEntry = mdEntries.Add()
	mdEntry.SetMDEntryType(enum.MDEntryType_TRADING_SESSION_LOW_PRICE)
	mdEntry.SetMDEntryPx(decimal.NewFromFloat(price.Low), 8)

	mdEntry = mdEntries.Add()
	mdEntry.SetMDEntryType(enum.MDEntryType_OPENING_PRICE)
	mdEntry.SetMDEntryPx(decimal.NewFromFloat(price.Open), 8)

	mdEntry = mdEntries.Add()
	mdEntry.SetMDEntryType(enum.MDEntryType_CLOSING_PRICE)
	mdEntry.SetMDEntryPx(decimal.NewFromFloat(price.Close), 8)

	/*
		mdEntry = mdEntries.Add()
		mdEntry.SetMDEntryPositionNo(1)
		mdEntry.SetMDEntryType(enum.MDEntryType_BID)
		mdEntry.SetMDEntryPx(decimal.NewFromFloat(price.Bid), 8)
		mdEntry.SetMDEntrySize(decimal.NewFromFloat(price.BidSize), 8)

		mdEntry = mdEntries.Add()
		mdEntry.SetMDEntryPositionNo(1)
		mdEntry.SetMDEntryType(enum.MDEntryType_OFFER)
		mdEntry.SetMDEntryPx(decimal.NewFromFloat(price.Ask), 8)
		mdEntry.SetMDEntrySize(decimal.NewFromFloat(price.AskSize), 8)*/

	mdEntry = mdEntries.Add()
	mdEntry.SetMDEntryType(enum.MDEntryType_TRADE)
	mdEntry.SetMDEntryPx(decimal.NewFromFloat(price.Last), 8)
	mdEntry.SetMDEntrySize(decimal.NewFromFloat(price.Volume), 8)
	msg.SetNoMDEntries(mdEntries)

	msg.SetNoMDEntries(mdEntries)
	for _, sessionID := range a.marketdataSubscribers {
		quickfix.SendToTarget(msg, sessionID)
	}
}

func (a *Application) BroadcastDepth(dom *orderbook.Depth) {
	depth, err := dom.Retrieve()
	if err != nil {
		return
	}

	symbol := a.pairFormater.Format(depth.Pair)
	msg := marketdatasnapshotfullrefresh.New(field.NewSymbol(symbol))
	msg.SetSymbol(symbol)
	msg.SetSecurityExchange(depth.Exchange)
	msg.SetSecurityType(ToSecurityType(depth.Asset))
	mdEntries := marketdatasnapshotfullrefresh.NewNoMDEntriesRepeatingGroup()

	const MAXDEPTH = 28

	for i := range depth.Bids {
		entry := depth.Bids[i]
		mdEntry := mdEntries.Add()
		mdEntry.SetMDEntryPositionNo(i + 1)
		mdEntry.SetMDEntryType(enum.MDEntryType_BID)
		mdEntry.SetMDEntryPx(decimal.NewFromFloat(entry.Price), 8)
		mdEntry.SetMDEntrySize(decimal.NewFromFloat(entry.Amount), 8)

		if i > MAXDEPTH {
			break
		}
	}

	for i := range depth.Asks {
		entry := depth.Asks[i]
		mdEntry := mdEntries.Add()
		mdEntry.SetMDEntryPositionNo(i + 1)
		mdEntry.SetMDEntryType(enum.MDEntryType_OFFER)
		mdEntry.SetMDEntryPx(decimal.NewFromFloat(entry.Price), 8)
		mdEntry.SetMDEntrySize(decimal.NewFromFloat(entry.Amount), 8)

		if i > MAXDEPTH {
			break
		}
	}

	msg.SetNoMDEntries(mdEntries)
	for _, sessionID := range a.marketdataSubscribers {
		quickfix.SendToTarget(msg, sessionID)
	}
}

func ToSide(side order.Side) enum.Side {
	switch side {
	case order.Buy:
		return enum.Side_BUY
	case order.Sell:
		return enum.Side_SELL
	default:
		return enum.Side_AS_DEFINED
	}
}

func (a *Application) SendFill(fills []fill.Data) {
	/*
		for i := range fills {
			fill := fills[i]

			msg, err := a.orderManager.orderStore.getByExchangeAndID(fill.Exchange, fill.OrderID)
			if err != nil {
				continue
			}

			symbol := a.pairFormater.Format(msg.Pair)
			status := ToOrdStatus(msg.Status)

			execReport := executionreport.New(
				field.NewOrderID(msg.OrderID),
				field.NewExecID(a.genExecID()),
				field.NewExecTransType(enum.ExecTransType_NEW),
				field.NewExecType(enum.ExecType_TRADE),
				field.NewOrdStatus(status),
				field.NewSymbol(symbol),
				field.NewSide(ToSide(msg.Side)),
				field.NewLeavesQty(decimal.NewFromFloat(msg.RemainingAmount), 8),
				field.NewCumQty(decimal.NewFromFloat(msg.ExecutedAmount), 8),
				field.NewAvgPx(decimal.NewFromFloat(msg.AverageExecutedPrice), 8),
			)

			if msg.Type == order.Limit || msg.Type == order.Stop {
				execReport.SetPrice(decimal.NewFromFloat(msg.Price), 8)
			} else if msg.Type == order.StopLimit {
				execReport.SetPrice(decimal.NewFromFloat(msg.Price), 8)
				execReport.SetStopPx(decimal.NewFromFloat(msg.TriggerPrice), 8)
			}

			execReport.SetOrderQty(decimal.NewFromFloat(msg.Amount), 8)
			execReport.SetClOrdID(msg.ClientOrderID)
			execReport.SetSecurityExchange(msg.Exchange)
			execReport.SetSecurityType(ToSecurityType(msg.AssetType))

			execReport.SetExecRefID(fill.TradeID)
			execReport.SetLastPx(decimal.NewFromFloat(fill.Price), 8)
			execReport.SetLastShares(decimal.NewFromFloat(fill.Amount), 8)

			for _, sessionID := range a.sessions {
				sendErr := quickfix.SendToTarget(execReport, sessionID)
				if sendErr != nil {
					fmt.Println(sendErr)
				}
			}
		}*/
}

func (a *Application) RejectOrderRequest(msg *order.Submit, text string) {
	symbol := a.pairFormater.Format(msg.Pair)
	execReport := executionreport.New(
		field.NewOrderID(a.genUUID()),
		field.NewExecID(a.genExecID()),
		field.NewExecTransType(enum.ExecTransType_NEW),
		field.NewExecType(enum.ExecType_REJECTED),
		field.NewOrdStatus(enum.OrdStatus_REJECTED),
		field.NewSymbol(symbol),
		field.NewSide(ToSide(msg.Side)),
		field.NewLeavesQty(decimal.NewFromFloat(0), 8),
		field.NewCumQty(decimal.NewFromFloat(0), 8),
		field.NewAvgPx(decimal.NewFromFloat(0), 8),
	)

	execReport.SetText(text)
	if msg.Type == order.Limit || msg.Type == order.Stop {
		execReport.SetPrice(decimal.NewFromFloat(msg.Price), 8)
	} else if msg.Type == order.StopLimit {
		execReport.SetPrice(decimal.NewFromFloat(msg.Price), 8)
		execReport.SetStopPx(decimal.NewFromFloat(msg.TriggerPrice), 8)
	}

	execReport.SetOrderQty(decimal.NewFromFloat(msg.Amount), 8)
	execReport.SetClOrdID(msg.ClientOrderID)
	execReport.SetSecurityExchange(msg.Exchange)
	execReport.SetSecurityType(ToSecurityType(msg.AssetType))

	for _, sessionID := range a.sessions {
		sendErr := quickfix.SendToTarget(execReport, sessionID)
		if sendErr != nil {
			fmt.Println(sendErr)
		}
	}
}

func (a *Application) UpdateOrder(msg *order.Detail, status enum.OrdStatus) {
	symbol := a.pairFormater.Format(msg.Pair)
	execReport := executionreport.New(
		field.NewOrderID(msg.OrderID),
		field.NewExecID(a.genExecID()),
		field.NewExecTransType(enum.ExecTransType_NEW),
		field.NewExecType(enum.ExecType(status)),
		field.NewOrdStatus(status),
		field.NewSymbol(symbol),
		field.NewSide(ToSide(msg.Side)),
		field.NewLeavesQty(decimal.NewFromFloat(msg.RemainingAmount), 8),
		field.NewCumQty(decimal.NewFromFloat(msg.ExecutedAmount), 8),
		field.NewAvgPx(decimal.NewFromFloat(msg.AverageExecutedPrice), 8),
	)

	if msg.Type == order.Limit || msg.Type == order.Stop {
		execReport.SetPrice(decimal.NewFromFloat(msg.Price), 8)
	} else if msg.Type == order.StopLimit {
		execReport.SetPrice(decimal.NewFromFloat(msg.Price), 8)
		execReport.SetStopPx(decimal.NewFromFloat(msg.TriggerPrice), 8)
	}

	execReport.SetOrderQty(decimal.NewFromFloat(msg.Amount), 8)
	execReport.SetClOrdID(msg.ClientOrderID)
	execReport.SetSecurityExchange(msg.Exchange)
	execReport.SetSecurityType(ToSecurityType(msg.AssetType))

	switch status {
	case enum.OrdStatus_FILLED, enum.OrdStatus_PARTIALLY_FILLED:
		execReport.SetExecType(enum.ExecType_TRADE)
		execReport.SetLastPx(decimal.NewFromFloat(msg.LastExecutedPrice), 8)
		execReport.SetLastShares(decimal.NewFromFloat(msg.LastExecutedQuantity), 8)
	}

	for _, sessionID := range a.sessions {
		sendErr := quickfix.SendToTarget(execReport, sessionID)
		if sendErr != nil {
			fmt.Println(sendErr)
		}
	}
}

func (a *Application) genExecID() string {
	a.execID++
	return strconv.Itoa(a.execID)
}

func (a *Application) genUUID() string {
	id, e := uuid.NewV4()
	if e != nil {
		return ""
	}
	return id.String()
}

func (a *Application) acceptOrder(order *order.Detail) {
	a.UpdateOrder(order, enum.OrdStatus_NEW)
}

func (a *Application) fillOrder(order *order.Detail) {
	status := enum.OrdStatus_FILLED
	if order.RemainingAmount > 0 {
		status = enum.OrdStatus_PARTIALLY_FILLED
	}
	a.UpdateOrder(order, status)
}

func (a *Application) cancelOrder(order *order.Detail) {
	a.UpdateOrder(order, enum.OrdStatus_CANCELED)
}
