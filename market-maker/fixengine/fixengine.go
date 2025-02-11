package fixengine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	"github.com/quickfixgo/fix42/executionreport"
	"github.com/quickfixgo/fix42/newordersingle"
	"github.com/quickfixgo/fix42/ordercancelreplacerequest"
	"github.com/quickfixgo/fix42/ordercancelrequest"
	"github.com/quickfixgo/fix42/securitydefinition"
	"github.com/quickfixgo/fix42/securitydefinitionrequest"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/quickfix/store/file"
	"github.com/quickfixgo/tag"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
	model "github.com/thrasher-corp/gocryptotrader/market-maker/models"
	"gopkg.in/ini.v1"
)

const (
	CCX = "CCX"
)

type SecurityDetail struct {
	Pair               currency.Pair
	ContractMultiplier float64
	PriceMultiplier    float64
}

type FixEngine struct {
	*quickfix.MessageRouter
	Username      string
	Password      string
	senderCompId  string
	targetCompId  string
	accountCode   string
	pairFormatter *currency.PairFormat
	initiator     *quickfix.Initiator
	settings      *quickfix.Settings
	logFactory    *quickfix.LogFactory
	storeFactory  quickfix.MessageStoreFactory
}

func NewFixEngine() *FixEngine {
	app := &FixEngine{
		MessageRouter: quickfix.NewMessageRouter(),
	}
	log.Infoln(log.FIXSys, "Fix engine initiated")
	app.AddRoute(executionreport.Route(app.onExecutionReport))
	app.AddRoute(securitydefinition.Route(app.onSecurityDefinition))
	return app
}

func (fe *FixEngine) OnCreate(sessionID quickfix.SessionID) {}

func (fe *FixEngine) OnLogon(sessionID quickfix.SessionID) {
	log.Infoln(log.FIXSys, "connected!")
}

func (fe *FixEngine) OnLogout(sessionID quickfix.SessionID) {}

func (fe *FixEngine) ToAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) {
	msgType, _ := msg.Header.GetString(tag.MsgType)

	if msgType == string(enum.MsgType_LOGON) {
		msg.Body.Set(field.NewUsername(fe.Username))
		msg.Body.Set(field.NewPassword(fe.Password))
	}
}

func (fe *FixEngine) ToApp(msg *quickfix.Message, sessionID quickfix.SessionID) error {
	return nil
}

func (fe *FixEngine) FromAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	msgType, err := msg.Header.GetString(tag.MsgType)
	if err != nil {
		log.Errorf(log.FIXSys, "received message error: %+v", err)
		return err
	}
	log.Debugf(log.FIXSys, "FromAdmin msg type: %s", msgType)
	return nil
}

func (fe *FixEngine) FromApp(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	log.Infof(log.FIXSys, "received message: %s", msg.String())
	return fe.Route(msg, sessionID)
}

func (fe *FixEngine) onSecurityDefinition(msg securitydefinition.SecurityDefinition, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	// symbol, err := msg.GetSymbol()
	// if err != nil {
	// 	return err
	// }
	// contractMultiplier, err := msg.GetContractMultiplier()
	// if err != nil {
	// 	return err
	// }
	// priceMultiplier, err := msg.Body.GetString(tag.TickIncrement)
	// if err != nil {
	// 	return err
	// }

	// if err := model.CheckExistingandAddPair(context.Background(), symbol, contractMultiplier.String(), priceMultiplier); err != nil {
	// 	log.Errorf(log.FIXSys, "error saving pair: %+v", err)
	// 	return nil
	// }
	return nil
}

func (fe *FixEngine) onExecutionReport(msg executionreport.ExecutionReport, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	orderDetail, err := ToOrderDetail(msg)
	if err != nil {
		return err
	}
	// delete order from redis if it's been cancelled
	switch orderDetail.Status {
	case order.Cancelled:
		if err := model.DeleteOrder(context.Background(), orderDetail.Copy()); err != nil {
			log.Errorf(log.FIXSys, "error when delete cancelled order: %+v", err)
			return nil
		}
		return nil
	case order.Filled:
		log.Debugf(log.FIXSys, "filled order: %+v", orderDetail)
		if err := AddCounterORderQueue(orderDetail.Copy()); err != nil {
			log.Errorf(log.FIXSys, "error when add counter order queue: %+v", err)
			return nil
		}
		if err := model.UpdateOrCreateOrder(orderDetail.Copy(), "filled order"); err != nil {
			log.Errorf(log.FIXSys, "error when updating the order: %+v", err)
			return nil
		}
		if err := model.DeleteOrder(context.Background(), orderDetail.Copy()); err != nil {
			log.Errorf(log.FIXSys, "error when delete filled order from redis: %+v", err)
			return nil
		}
		executedAmount, err := msg.GetLastShares()
		if err != nil {
			return err
		}
		lastPrice, err := msg.GetLastPx()
		if err != nil {
			return err
		}
		if !decimal.Zero.Equal(executedAmount) && !decimal.Zero.Equal(lastPrice) {
			trade := model.Trade{
				TradeID:   orderDetail.ClientID,
				OrderID:   orderDetail.OrderID,
				Exchange:  CCX,
				Price:     lastPrice.InexactFloat64(),
				Quantity:  executedAmount.InexactFloat64(),
				Timestamp: orderDetail.LastUpdated,
			}
			if err := model.UpdateOrCreateTrade(trade.TradeID, trade); err != nil {
				log.Errorf(log.FIXSys, "error when saved trade: %+v", err)
				return nil
			}
			return nil
		}
		return nil
	case order.PartiallyFilled:
		log.Debugf(log.FIXSys, "partially filled order: %+v", orderDetail)
		if err := fe.CancelOrder(*orderDetail); err != nil {
			log.Errorf(log.FIXSys, "failed to cancel partial filled order: %+v", err)
			return nil
		}
		executedAmount, err := msg.GetLastShares()
		if err != nil {
			return err
		}
		lastPrice, err := msg.GetLastPx()
		if err != nil {
			return err
		}
		if !decimal.Zero.Equal(executedAmount) && !decimal.Zero.Equal(lastPrice) {
			orderDetail.Price = lastPrice.InexactFloat64()
			orderDetail.Amount = executedAmount.InexactFloat64()
			if err := AddCounterORderQueue(orderDetail.Copy()); err != nil {
				log.Errorf(log.FIXSys, "error when add counter order queue: %+v", err)
				return nil
			}
			trade := model.Trade{
				TradeID:   orderDetail.ClientID,
				OrderID:   orderDetail.OrderID,
				Exchange:  orderDetail.Exchange,
				Price:     lastPrice.InexactFloat64(),
				Quantity:  executedAmount.InexactFloat64(),
				Timestamp: orderDetail.LastUpdated,
			}
			if err := model.UpdateOrCreateTrade(trade.TradeID, trade); err != nil {
				log.Errorf(log.FIXSys, "error when create trade: %+v", err)
				return nil
			}
			return nil
		}
	case order.Rejected:
		description, _ := msg.Body.GetString(tag.Text)
		orderDetail.OrderID = fmt.Sprintf("%d-%s", time.Now().Unix(), generateRandomString(7))
		if err := model.UpdateOrCreateOrder(orderDetail.Copy(), description); err != nil {
			log.Errorf(log.FIXSys, "error when save rejected order: %+v", err)
			return nil
		}
		return nil
	case order.New:
		if err := model.UpdateOrCreateOrderRedis(context.Background(), orderDetail.Copy()); err != nil {
			log.Errorf(log.FIXSys, "error when updating the order: %+v", err)
			return nil
		}
		return nil
	default:
		log.Errorf(log.FIXSys, "invalid order status: %+v", orderDetail)
		return nil
	}
	return nil
}

func AddCounterORderQueue(orderDetail order.Detail) error {
	switch orderDetail.Side {
	case order.Buy:
		orderDetail.Side = order.Sell
	case order.Sell:
		orderDetail.Side = order.Buy
	default:
		return order.ErrSideIsInvalid
	}

	orderDetail.Type = order.Market
	if orderDetail.ClientID != "" {
		orderDetail.ClientOrderID = orderDetail.ClientID
	}
	return model.AddCounterOrderQueue(context.Background(), orderDetail)
}

func (fe *FixEngine) Start() error {
	var cfgFileName string
	fileName := "market-maker.cfg"
	cfgFileName = path.Join("./", fileName)

	cfg, err := os.Open(cfgFileName)
	if err != nil {
		return err
	}
	defer cfg.Close()
	stringData, err := io.ReadAll(cfg)
	if err != nil {
		return err
	}

	config, err := ini.Load(cfgFileName)
	if err != nil {
		return fmt.Errorf("error reading cfg: %s,", err)
	}
	fe.senderCompId = config.Section("DEFAULT").Key("SenderCompID").String()
	fe.targetCompId = config.Section("SESSION").Key("TargetCompID").String()
	fe.accountCode = config.Section("SESSION").Key("AccountCode").String()

	fe.settings, err = quickfix.ParseSettings(bytes.NewReader(stringData))
	if err != nil {
		return fmt.Errorf("error reading setting cfg: %+v", err)
	}

	fe.storeFactory = file.NewStoreFactory(fe.settings)
	logFactory, err := quickfix.NewFileLogFactory(fe.settings)
	if err != nil {
		return fmt.Errorf("unable to create logger: %s", err)
	}
	fe.logFactory = &logFactory

	fe.pairFormatter = &currency.PairFormat{
		Uppercase: true,
		Delimiter: "-",
	}

	fe.Username = config.Section("SESSION").Key("UserName").String()
	fe.Password = config.Section("SESSION").Key("Password").String()
	initiator, err := quickfix.NewInitiator(fe, fe.storeFactory, fe.settings, *fe.logFactory)
	if err != nil {
		return fmt.Errorf("error when initiate initiator : %+v", err)
	}
	fe.initiator = initiator
	return fe.initiator.Start()
}

func (fe *FixEngine) Stop() {
	fe.initiator.Stop()
}

func (fe *FixEngine) SecuritiesDetail() error {
	securityDefinitionRequest := securitydefinitionrequest.New(
		field.NewSecurityReqID("1"),
		field.NewSecurityRequestType(enum.SecurityRequestType_REQUEST_LIST_SECURITIES),
	)
	securityDefinitionRequest.Set(field.NewSecurityExchange("CCX"))
	sdrMsg := securityDefinitionRequest.ToMessage()
	sdrMsg.Header.Set(field.NewSenderCompID(fe.senderCompId))
	sdrMsg.Header.Set(field.NewTargetCompID(fe.targetCompId))

	return quickfix.Send(sdrMsg)
}

func (fe *FixEngine) NewOrderSingle(order order.Detail) error {
	if !fe.initiator.BuildInitiators {
		return nil
	}
	newOrder := newordersingle.New(
		field.NewClOrdID(GenerateClOrdID()),
		field.NewHandlInst(enum.HandlInst_AUTOMATED_EXECUTION_ORDER_PRIVATE_NO_BROKER_INTERVENTION),
		field.NewSymbol(order.Pair.Base.String()),
		field.NewSide(convertSide(order.Side.String())),
		field.NewTransactTime(time.Now().UTC()),
		field.NewOrdType(convertOrdType(order.Type.String())),
	)

	newOrder.Set(field.NewAccount(fe.accountCode))
	newOrder.Set(field.NewSecurityType(convertAsset(order.AssetType.String())))
	newOrder.Set(field.NewSecurityExchange(order.Exchange))
	newOrder.Set(field.NewTimeInForce(convertTIF("DAY")))
	newOrder.Set(field.NewPrice(decimal.NewFromFloat(order.Price), 8))
	newOrder.Set(field.NewOrderQty(decimal.NewFromFloat(order.Amount), 8))
	orderMsg := newOrder.ToMessage()
	orderMsg.Header.Set(field.NewSenderCompID(fe.senderCompId))
	orderMsg.Header.Set(field.NewTargetCompID(fe.targetCompId))

	return quickfix.Send(orderMsg)
}

func (fe *FixEngine) CancelOrder(order order.Detail) error {
	cancelReq := ordercancelrequest.New(
		field.NewOrigClOrdID(order.ClientOrderID),
		field.NewClOrdID(GenerateClOrdID()),
		field.NewSymbol(order.Pair.Base.String()),
		field.NewSide(convertSide(order.Side.String())),
		field.NewTransactTime(time.Now().UTC()),
	)
	cancelReq.SetOrderID(order.OrderID)
	cancelReq.SetSecurityType(convertAsset(order.AssetType.String()))
	cancelReq.SetSecurityExchange(order.Exchange)
	cancelReq.SetAccount(fe.accountCode)
	cancelMsg := cancelReq.ToMessage()
	cancelMsg.Header.Set(field.NewSenderCompID(fe.senderCompId))
	cancelMsg.Header.Set(field.NewTargetCompID(fe.targetCompId))
	return quickfix.Send(cancelMsg)
}

func (fe *FixEngine) CancelReplaceOrder(order order.Detail) error {
	modifyReq := ordercancelreplacerequest.New(
		field.NewOrigClOrdID(order.ClientOrderID),
		field.NewClOrdID(GenerateClOrdID()),
		field.NewHandlInst(enum.HandlInst_AUTOMATED_EXECUTION_ORDER_PRIVATE_NO_BROKER_INTERVENTION),
		field.NewSymbol(order.Pair.Base.String()),
		field.NewSide(convertSide(order.Side.String())),
		field.NewTransactTime(time.Now().UTC()),
		field.NewOrdType(convertOrdType(order.Type.String())),
	)

	modifyReq.SetOrderID(order.OrderID)
	modifyReq.SetSecurityType(convertAsset(order.AssetType.String()))
	modifyReq.SetSecurityExchange(order.Exchange)
	modifyReq.SetPrice(decimal.NewFromFloat(order.Price), 8)
	modifyReq.SetOrderQty(decimal.NewFromFloat(order.Amount), 8)
	modifyReq.SetAccount(fe.accountCode)
	modifyMsg := modifyReq.ToMessage()
	modifyMsg.Header.Set(field.NewSenderCompID(fe.senderCompId))
	modifyMsg.Header.Set(field.NewTargetCompID(fe.targetCompId))
	return quickfix.Send(modifyMsg)
}

func (fe *FixEngine) GetCCXPairs() ([]SecurityDetail, error) {
	var ccxPairs []SecurityDetail
	pairs, err := model.GetPairs(context.Background())
	if err != nil {
		return nil, err
	}
	if len(pairs) == 0 {
		return ccxPairs, nil
	}
	for i := range pairs {
		switch pairs[i].Base {
		// only available for this base
		case "AVAX", "BCH", "BTC", "BNB", "ETH", "LTC", "SOL":
			ccxPairs = append(ccxPairs, SecurityDetail{
				Pair:               currency.NewPairWithDelimiter(pairs[i].Base, pairs[i].Quote, pairs[i].Delimiter),
				ContractMultiplier: pairs[i].ContractMultiplier,
				PriceMultiplier:    pairs[i].PriceIncrement,
			})
		}
	}
	return ccxPairs, nil
}
