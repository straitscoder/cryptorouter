package tsclient

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/apache/thrift/lib/go/thrift"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/log"
	model "github.com/thrasher-corp/gocryptotrader/market-maker/models"
	"github.com/thrasher-corp/gocryptotrader/market-maker/trading-service/tradingservice"
	"github.com/thrasher-corp/gocryptotrader/market-maker/trading-service/types"
)

const (
	ccx = "CCX"
)

type SecurityDetail struct {
	Pair               currency.Pair
	ContractMultiplier float64
	PriceMultiplier    float64
}

type TSClient struct {
	processOrder int32
	username     string
	password     string
	host         string
	port         int32
	beginIndex   int32
	Account      *types.Account
}

func NewTSClient(username, password, host string, port int) *TSClient {
	return &TSClient{
		username: username,
		password: password,
		host:     host,
		port:     int32(port),
	}
}

func (ts *TSClient) getClient() (*tradingservice.TradingServiceClient, error) {
	addr := fmt.Sprintf("%s:%d", ts.host, ts.port)
	conf := &thrift.TConfiguration{
		TBinaryStrictRead:  thrift.BoolPtr(true),
		TBinaryStrictWrite: thrift.BoolPtr(true),
	}

	socket := thrift.NewTSocketConf(addr, conf)
	transportFactory := thrift.NewTFramedTransportFactoryConf(thrift.NewTTransportFactory(), conf)
	protoFactory := thrift.NewTBinaryProtocolFactoryConf(conf)

	transport, err := transportFactory.GetTransport(socket)
	if err != nil {
		return nil, err
	}

	err = transport.Open()
	if err != nil {
		return nil, err
	}

	return tradingservice.NewTradingServiceClientFactory(transport, protoFactory), nil
}

func (ts *TSClient) Start() error {
	account, err := ts.GetDefaultAccount()
	if err != nil {
		return err
	}

	ts.Account = account
	return nil
}

func (ts *TSClient) GetDefaultAccount() (*types.Account, error) {
	client, err := ts.getClient()
	if err != nil {
		return nil, err
	}

	account, err := client.GetDefaultAccount(context.Background(), ts.username)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.New("Invalid account")
	}
	return account, nil
}

func (ts *TSClient) NewOrder(ctx context.Context, o order.Detail) error {
	if !atomic.CompareAndSwapInt32(&ts.processOrder, 0, 1) {
		return nil
	}
	defer atomic.StoreInt32(&ts.processOrder, 0)
	client, err := ts.getClient()
	if err != nil {
		return err
	}
	instrument, err := model.GetPair(ctx, o.Pair.Base.String())
	if err != nil {
		return err
	}
	if instrument == nil {
		return errors.New("invalid instrumment")
	}
	req := types.Order{
		Account: ts.username,
		User:    ts.username,
		// ClOrdID:    ts.generateRandomInt(),
		Exchange:     o.Exchange,
		Side:         toTSSide(o.Side),
		OrderType:    toTSType(o.Type),
		InstrumentID: instrument.InstrumentID,
		Instrument:   instrument.Base,
		Price:        o.Price,
		Qty:          int32(o.Amount),
		IpAddr:       &ts.host,
		Port:         &ts.port,
	}
	result, err := client.PlaceOrder(ctx, &req)
	if err != nil {
		return err
	}

	if types.ErrorCode(result).String() != "OK" {
		return errors.New(types.ErrorCode(result).String())
	}
	return nil
}

func (ts *TSClient) CancelOrder(ctx context.Context, o order.Detail) error {
	if !atomic.CompareAndSwapInt32(&ts.processOrder, 0, 1) {
		return nil
	}
	defer atomic.StoreInt32(&ts.processOrder, 0)
	client, err := ts.getClient()
	if err != nil {
		return err
	}
	orderID, err := strconv.Atoi(o.OrderID)
	if err != nil {
		return err
	}
	instrument, err := model.GetPair(ctx, o.Pair.Base.String())
	if err != nil {
		return err
	}
	if instrument == nil {
		return errors.New("invalid instrument")
	}
	req := types.Order{
		Account:      ts.username,
		ID:           int32(orderID),
		Exchange:     o.Exchange,
		Instrument:   instrument.Base,
		InstrumentID: instrument.InstrumentID,
		Side:         toTSSide(o.Side),
		OrderType:    toTSType(o.Type),
		Price:        o.Price,
		Qty:          int32(o.Amount),
		IpAddr:       &ts.host,
		Port:         &ts.port,
	}

	result, err := client.CancelOrder(ctx, &req)
	if err != nil {
		return err
	}

	if types.ErrorCode(result).String() != "OK" {
		return errors.New(types.ErrorCode(result).String())
	}
	return nil
}

func (ts *TSClient) GetOrders(ctx context.Context, criteria *types.SearchCriteria) ([]order.Detail, error) {
	if criteria == nil {
		return nil, errors.New("criteria is nil")
	}
	client, err := ts.getClient()
	if err != nil {
		return nil, err
	}
	if criteria.Instrument != nil {
		instrument, err := model.GetPair(ctx, *criteria.Instrument)
		if err != nil {
			return nil, err
		}

		if instrument == nil {
			return nil, errors.New("invalid instrument")
		}
		criteria.Instrument = &instrument.Base
		criteria.InstrumentID = &instrument.InstrumentID
	}
	criteria.AccountID = ts.Account.ID
	index := GetIndex(*criteria.Instrument)
	criteria.BeginIndex = &index

	tsOrders, err := client.GetOrders(ctx, criteria)
	if err != nil {
		return nil, err
	}

	var orders []order.Detail
	if len(tsOrders) > 0 {
		for i := range tsOrders {
			SaveIndex(tsOrders[i].Instrument, tsOrders[i].LastMsgID)
			orderDetail, err := toOrderDetail(tsOrders[i])
			if err != nil {
				return nil, err
			}
			orders = append(orders, orderDetail)
		}
	}

	return orders, nil
}

func (ts *TSClient) GetOrder(ctx context.Context, orderID string) (order.Detail, error) {
	client, err := ts.getClient()
	if err != nil {
		return order.Detail{}, err
	}

	ordIDInt, err := strconv.Atoi(orderID)
	if err != nil {
		return order.Detail{}, err
	}

	tsOrder, err := client.GetOrder(ctx, int32(ordIDInt))
	if err != nil {
		return order.Detail{}, err
	}

	return toOrderDetail(tsOrder)
}

func (ts *TSClient) GetPairs(ctx context.Context, exchange string) error {
	client, err := ts.getClient()
	if err != nil {
		return err
	}

	pairs, err := client.GetInstruments(ctx, &types.SearchCriteria{
		Exchange: &exchange,
		Account:  &ts.username,
	})
	if len(pairs) > 0 {
		for i := range pairs {
			pair := model.Pair{
				InstrumentID:       pairs[i].GetID(),
				Base:               pairs[i].Code,
				ContractMultiplier: pairs[i].TickValue,
				PriceIncrement:     pairs[i].TickSize,
			}
			log.Infof(log.GRPCSys, "available pair: %+v", pair)
			if err := model.CheckExistingandAddPair(
				ctx,
				pairs[i].GetID(),
				pairs[i].Code,
				pairs[i].TickValue,
				pairs[i].TickSize,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (ts *TSClient) GetCCXPairs() ([]SecurityDetail, error) {
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

func toTSSide(side order.Side) types.Side {
	switch side {
	case order.Buy:
		return types.Side_BUY
	case order.Sell:
		return types.Side_SELL
	default:
		return types.Side_SELL
	}
}

func toTSType(t order.Type) types.OrderType {
	switch t {
	case order.Limit:
		return types.OrderType_LIMIT
	case order.Market:
		return types.OrderType_MARKET
	default:
		return types.OrderType_STOP
	}
}

func toOrderDetail(o *types.Order) (order.Detail, error) {
	if o == nil {
		return order.Detail{}, nil
	}
	pair := currency.NewPairWithDelimiter(o.Instrument, "USD", "-")
	orderType, err := order.StringToOrderType(o.OrderType.String())
	if err != nil {
		return order.Detail{}, err
	}
	side, err := order.StringToOrderSide(o.Side.String())
	if err != nil {
		return order.Detail{}, err
	}
	zeroFloat := float64(0)
	zeroInt := int32(0)
	var trades []order.TradeHistory
	if o.LastPx != nil && o.LastShares != nil && o.LastPx != &zeroFloat && o.LastShares != &zeroInt {
		trades = append(trades, order.TradeHistory{
			Price:     *o.LastPx,
			Amount:    float64(*o.LastShares),
			Exchange:  ccx,
			TID:       strconv.FormatInt(int64(o.LastMsgID), 10),
			Type:      orderType,
			Side:      side,
			Timestamp: time.UnixMilli(o.TransactTime),
			Total:     float64(o.CumQty),
		})
	}
	return order.Detail{
		Exchange:             ccx,
		AssetType:            asset.Futures,
		OrderID:              strconv.FormatInt(int64(o.ID), 10),
		ClientOrderID:        strconv.FormatInt(int64(o.LastMsgID), 10),
		Type:                 orderType,
		Side:                 side,
		Status:               toStatus(o.Status),
		Price:                o.Price,
		AverageExecutedPrice: o.AvgPrice,
		Amount:               float64(o.Qty),
		ExecutedAmount:       float64(o.CumQty),
		RemainingAmount:      float64(o.LeavesQty),
		Pair:                 pair,
		Date:                 time.UnixMilli(o.TransactTime),
		Trades:               trades,
	}, nil
}

func toStatus(status types.OrderStatus) order.Status {
	switch status {
	case types.OrderStatus_NEW:
		return order.New
	case types.OrderStatus_CANCELED:
		return order.Cancelled
	case types.OrderStatus_FILLED:
		return order.Filled
	case types.OrderStatus_PARTIALLY_FILLED:
		return order.PartiallyFilled
	case types.OrderStatus_PENDING_NEW:
		return order.Pending
	case types.OrderStatus_PENDING_CANCEL:
		return order.PendingCancel
	default:
		return order.Rejected
	}
}
