package fixengine

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"sync/atomic"
	"time"

	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	"github.com/quickfixgo/fix42/newordersingle"
	"github.com/quickfixgo/fix42/ordercancelreplacerequest"
	"github.com/quickfixgo/fix42/ordercancelrequest"
	"github.com/quickfixgo/fix42/securitydefinitionrequest"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/quickfix/store/file"
	"github.com/quickfixgo/tag"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	model "github.com/thrasher-corp/gocryptotrader/market-maker/models"
	"gopkg.in/ini.v1"
)

const (
	CCX = "CCX"
)

var seqNum int

type SecurityDetail struct {
	Pair               currency.Pair
	ContractMultiplier float64
	PriceMultiplier    float64
}

type fixApplication struct {
	Username string
	Password string
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
	case "d":
		symbol, _ := msg.Body.GetString(tag.Symbol)
		contractMultiplier, _ := msg.Body.GetString(tag.ContractMultiplier)
		priceIncrement, _ := msg.Body.GetString(tag.TickIncrement)
		if err := model.CheckExistingandAddPair(context.Background(), symbol, contractMultiplier, priceIncrement); err != nil {
			log.Print(err)
			return
		}
	case "8":
		ordStatus, _ := msg.Body.GetString(tag.OrdStatus)
		if ordStatus != "0" && ordStatus != "4" {
			log.Printf("execution report: %+v", msg)
		}
		orderDetail := ToOrderDetail(msg)
		// delete order from redis if it's been cancelled
		switch orderDetail.Status {
		case order.Cancelled:
			if err := model.DeleteOrder(context.Background(), orderDetail); err != nil {
				log.Printf("error when delete cancelled order: %+v", err)
				return nil
			}
			return nil
		case order.Filled:
			log.Printf("filled order: %+v", orderDetail)
			if err := c.AddCounterORderQueue(orderDetail); err != nil {
				log.Printf("error when add counter order queue: %+v", err)
				return nil
			}
			if err := model.UpdateOrCreateOrder(orderDetail, "filled order"); err != nil {
				log.Printf("error when updating the order: %+v", err)
				return nil
			}
			if err := model.DeleteOrder(context.Background(), orderDetail); err != nil {
				log.Printf("error when delete filled order from redis: %+v", err)
				return nil
			}
			executedAmountStr, _ := msg.Body.GetString(tag.LastQty)
			lastPriceStr, _ := msg.Body.GetString(tag.LastPx)
			if executedAmountStr != "0" && lastPriceStr != "0" {
				executedAmount, _ := decimal.NewFromString(executedAmountStr)
				lastPrice, _ := decimal.NewFromString(lastPriceStr)
				trade := model.Trade{
					TradeID:   orderDetail.ClientID,
					OrderID:   orderDetail.OrderID,
					Exchange:  CCX,
					Price:     lastPrice.InexactFloat64(),
					Quantity:  executedAmount.InexactFloat64(),
					Timestamp: orderDetail.LastUpdated,
				}
				if err := model.UpdateOrCreateTrade(trade.TradeID, trade); err != nil {
					log.Printf("error when saved trade: %+v", err)
					return nil
				}
				return nil
			}
			return nil
		case order.PartiallyFilled:
			log.Printf("partially filled order: %+v", orderDetail)
			if err := model.AddCancelQueue(context.Background(), orderDetail); err != nil {
				log.Printf("error when add cancel order queue: %+v", err)
				return
			}
			executedAmountStr, _ := msg.Body.GetString(tag.LastQty)
			lastPriceStr, _ := msg.Body.GetString(tag.LastPx)
			if executedAmountStr != "0" && lastPriceStr != "0" {
				executedAmount, _ := decimal.NewFromString(executedAmountStr)
				lastPrice, _ := decimal.NewFromString(lastPriceStr)
				orderDetail.Price = lastPrice.InexactFloat64()
				orderDetail.Amount = executedAmount.InexactFloat64()
				if err := c.AddCounterORderQueue(orderDetail); err != nil {
					log.Printf("error when add counter order queue: %+v", err)
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
					log.Printf("error when create trade: %+v", err)
					return nil
				}
				return nil
			}
		case order.Rejected:
			description, _ := msg.Body.GetString(tag.Text)
			orderDetail.OrderID = fmt.Sprintf("%d-%s", time.Now().Unix(), generateRandomString(7))
			if err := model.UpdateOrCreateOrder(orderDetail, description); err != nil {
				log.Printf("error when save rejected order: %+v", err)
				return nil
			}
			return nil
		case order.New:
			if err := model.UpdateOrCreateOrderRedis(context.Background(), orderDetail); err != nil {
				log.Printf("error when updating the order: %+v", err)
				return nil
			}
			return nil
		default:
			log.Printf("invalid order status: %+v", orderDetail)
			return nil
		}
	}
	return nil
}

func (c *fixApplication) ToAdmin(msg *quickfix.Message, sessionID quickfix.SessionID) {
	msgType, _ := msg.Header.GetString(tag.MsgType)

	if msgType == string(enum.MsgType_LOGON) {
		msg.Body.Set(field.NewUsername(c.Username))
		msg.Body.Set(field.NewPassword(c.Password))
	}
}

func (c *fixApplication) ToApp(msg *quickfix.Message, sessionID quickfix.SessionID) error {
	return nil
}

func (c *fixApplication) AddCounterORderQueue(orderDetail order.Detail) error {
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

type FixEngine struct {
	processOrder  int32
	senderCompId  string
	targetCompId  string
	accountCode   string
	pairFormatter *currency.PairFormat
	initiator     *quickfix.Initiator
	settings      *quickfix.Settings
	logFactory    *quickfix.LogFactory
	storeFactory  quickfix.MessageStoreFactory
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

	logFactory, err := quickfix.NewFileLogFactory(fe.settings)
	if err != nil {
		return fmt.Errorf("unable to create logger: %s", err)
	}
	fe.logFactory = &logFactory

	fe.storeFactory = file.NewStoreFactory(fe.settings)
	fe.pairFormatter = &currency.PairFormat{
		Uppercase: true,
		Delimiter: "-",
	}

	app := &fixApplication{}
	app.Username = config.Section("SESSION").Key("UserName").String()
	app.Password = config.Section("SESSION").Key("Password").String()
	initiator, err := quickfix.NewInitiator(app, fe.storeFactory, fe.settings, *fe.logFactory)
	if err != nil {
		return fmt.Errorf("error when initiate initiator : %+v", err)
	}
	fe.initiator = initiator
	go fe.CancelOrderRoutine()
	return fe.initiator.Start()
}

func (fe *FixEngine) Stop() {
	fe.initiator.Stop()
}

func (fe *FixEngine) CancelOrderRoutine() {
	if fe == nil {
		return
	}

	ticker := time.NewTicker(time.Millisecond * 500)
	defer ticker.Stop()
	fe.SendCancelOrder()

	for {
		select {
		case <-ticker.C:
			go fe.SendCancelOrder()
		}
	}
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

func (fe *FixEngine) SendCancelOrder() {
	if !atomic.CompareAndSwapInt32(&fe.processOrder, 0, 1) {
		return
	}
	defer atomic.StoreInt32(&fe.processOrder, 0)

	if !fe.initiator.BuildInitiators {
		return
	}

	orderDetail, err := model.GetCancelQueue(context.Background())
	if err != nil {
		log.Printf("error when get counter order queue: %+v", err)
		return
	}
	if orderDetail.OrderID == "" {
		return
	}
	if err := fe.CancelOrder(orderDetail); err != nil {
		log.Printf("error when send counter order queue: %+v", err)
		if err := model.AddCancelQueue(context.Background(), orderDetail); err != nil {
			log.Printf("error when resaved error counter order: %+v", err)
			return
		}
		return
	}
	return
}
