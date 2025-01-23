package fixengine

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func closeConn(conn *quickfix.Initiator, cancel context.CancelFunc) {
	conn.Stop()
	if cancel != nil {
		cancel()
	}
}

// TODO: create function to generate client order id in 6 digits with prefix for quickfix
func generateRandomString(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		panic(err) // Handle error appropriately in production
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

func generateClOrdID() string {
	timestamp := time.Now().Unix()         // Unix timestamp for uniqueness
	randomPart := generateRandomString(10) // Random alphanumeric string
	clOrdId := fmt.Sprintf("%s%s", strconv.FormatInt(timestamp, 10)[:5], randomPart)
	if len(clOrdId) > 8 {
		clOrdId = clOrdId[:8]
	}
	return clOrdId
}

func parseFIXMessage(msg *quickfix.Message) map[string]interface{} {
	parsed := make(map[string]interface{})
	addFieldsToMap(parsed, &msg.Header.FieldMap)
	addFieldsToMap(parsed, &msg.Body.FieldMap)
	addFieldsToMap(parsed, &msg.Trailer.FieldMap)
	return parsed
}

func addFieldsToMap(m map[string]interface{}, group *quickfix.FieldMap) {
	tags := group.Tags()
	for _, tag := range tags {
		value, _ := group.GetString(tag)
		fieldName := fmt.Sprintf("%d", tag)
		m[fieldName] = value
	}
}

func jsonOutput(in interface{}) {
	j, err := json.MarshalIndent(in, "", " ")
	if err != nil {
		return
	}
	fmt.Printf("%s\n", string(j))
}

func Confirmation() bool {
	fmt.Println("Are you sure? (Y/N)")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.ToUpper(scanner.Text()) == "Y"
}

func convertSide(side string) enum.Side {
	switch side {
	case "BUY":
		return enum.Side_BUY
	case "SELL":
		return enum.Side_SELL
	default:
		return enum.Side_AS_DEFINED
	}
}

func convertOrdType(ordType string) enum.OrdType {
	switch strings.ToUpper(ordType) {
	case "LIMIT":
		return enum.OrdType_LIMIT
	case "MARKET":
		return enum.OrdType_MARKET
	default:
		return enum.OrdType_LIMIT
	}
}

func convertAsset(orderAsset string) enum.SecurityType {
	switch strings.ToUpper(orderAsset) {
	case "SPOT":
		return enum.SecurityType_FX_SPOT
	case "FUTURE", asset.Futures.String(), "FUTURES":
		return enum.SecurityType_FUTURE
	default:
		return enum.SecurityType_FX_FORWARD
	}
}

func convertHandleInst(handleIns string) enum.HandlInst {
	switch strings.ToUpper(handleIns) {
	case "SEMI":
		return enum.HandlInst_AUTOMATED_EXECUTION_ORDER_PUBLIC_BROKER_INTERVENTION_OK
	case "MANUAL":
		return enum.HandlInst_MANUAL_ORDER_BEST_EXECUTION
	default:
		return enum.HandlInst_AUTOMATED_EXECUTION_ORDER_PRIVATE_NO_BROKER_INTERVENTION
	}
}

func convertSubsReqType(subsReqType string) enum.SubscriptionRequestType {
	switch strings.ToUpper(subsReqType) {
	case "SNAPSHOT":
		return enum.SubscriptionRequestType_SNAPSHOT
	case "SNAPSHOTPLUS":
		return enum.SubscriptionRequestType_SNAPSHOT_PLUS_UPDATES
	case "DISABLEPREVIOUS":
		return enum.SubscriptionRequestType_DISABLE_PREVIOUS_SNAPSHOT_PLUS_UPDATE_REQUEST
	default:
		return enum.SubscriptionRequestType_SNAPSHOT
	}
}

func convertMarketDepth(marketDepth string) int {
	switch strings.ToUpper(marketDepth) {
	case "TOPOFBOOK":
		return 1
	case "FULLBOOK":
		return 0
	default:
		return 0
	}
}

func convertMDUpdateType(mdUpdateType string) enum.MDUpdateType {
	switch strings.ToUpper(mdUpdateType) {
	case "FULLREFRESH":
		return enum.MDUpdateType_FULL_REFRESH
	case "INCREMENTALREFRESH":
		return enum.MDUpdateType_INCREMENTAL_REFRESH
	default:
		return enum.MDUpdateType_FULL_REFRESH
	}
}

func convertMDEntryType(mDEntryType string) enum.MDEntryType {
	switch strings.ToUpper(mDEntryType) {
	case "BID":
		return enum.MDEntryType_BID
	case "OFFER":
		return enum.MDEntryType_OFFER
	case "TRADE":
		return enum.MDEntryType_TRADE
	default:
		return enum.MDEntryType_TRADE
	}
}

func convertTIF(timeInForce string) enum.TimeInForce {
	switch strings.ToUpper(timeInForce) {
	case "GTC":
		return enum.TimeInForce_GOOD_TILL_CANCEL
	case "IOC":
		return enum.TimeInForce_IMMEDIATE_OR_CANCEL
	default:
		return enum.TimeInForce_DAY
	}
}

func ToStatus(ordStatus string) order.Status {
	switch ordStatus {
	case "0":
		return order.New
	case "1":
		return order.PartiallyFilled
	case "2":
		return order.Filled
	case "4":
		return order.Cancelled
	case "6":
		return order.PendingCancel
	case "A":
		return order.Pending
	default:
		return order.Rejected
	}
}

func ToType(ordType string) order.Type {
	switch ordType {
	case "1":
		return order.Market
	case "2":
		return order.Limit
	case "3":
		return order.Stop
	case "4":
		return order.Stop
	default:
		return order.UnknownType
	}
}

func ToSide(side string) order.Side {
	switch side {
	case "1":
		return order.Buy
	case "2":
		return order.Sell
	default:
		return order.AnySide
	}
}

func ToOrderDetail(msg *quickfix.Message) order.Detail {
	clOrdID, _ := msg.Body.GetString(tag.ClOrdID)
	orderId, _ := msg.Body.GetString(tag.OrderID)
	ordStatus, _ := msg.Body.GetString(tag.OrdStatus)
	exchange, _ := msg.Body.GetString(tag.SecurityExchange)
	ordType, _ := msg.Body.GetString(tag.OrdType)
	symbol, _ := msg.Body.GetString(tag.Symbol)
	side, _ := msg.Body.GetString(tag.Side)
	ordQty, _ := msg.Body.GetString(tag.OrderQty)
	price, _ := msg.Body.GetString(tag.Price)
	remainingQty, _ := msg.Body.GetString(tag.LeavesQty)
	filledQty, _ := msg.Body.GetString(tag.CumQty)
	avgPrice, _ := msg.Body.GetString(tag.AvgPx)
	timestamp, _ := msg.Body.GetTime(tag.TransactTime)
	orderDetail := order.Detail{
		AssetType: asset.Futures,
	}
	if clOrdID != "" {
		orderDetail.ClientOrderID = clOrdID
	}
	if orderId != "" {
		orderDetail.OrderID = orderId
	}
	if ordStatus != "" {
		orderDetail.Status = ToStatus(ordStatus)
	}
	if exchange != "" {
		orderDetail.Exchange = exchange
	}
	if ordType != "" {
		orderDetail.Type = ToType(ordType)
	}
	if symbol != "" {
		orderDetail.Pair = currency.NewPairWithDelimiter(symbol, "USD", "-")
	}
	if side != "" {
		orderDetail.Side = ToSide(side)
	}
	if ordQty != "" {
		amount, _ := decimal.NewFromString(ordQty)
		orderDetail.Amount = amount.InexactFloat64()
	}
	if price != "" {
		priceD, _ := decimal.NewFromString(price)
		orderDetail.Price = priceD.InexactFloat64()
	}
	if remainingQty != "" {
		remainingAmount, _ := decimal.NewFromString(remainingQty)
		orderDetail.RemainingAmount = remainingAmount.InexactFloat64()
	}
	if filledQty != "" {
		FilledAmount, _ := decimal.NewFromString(filledQty)
		orderDetail.ExecutedAmount = FilledAmount.InexactFloat64()
	}
	if avgPrice != "" {
		avgPx, _ := decimal.NewFromString(avgPrice)
		orderDetail.AverageExecutedPrice = avgPx.InexactFloat64()
	}
	if !timestamp.IsZero() {
		orderDetail.LastUpdated = timestamp
	}
	return orderDetail
}

var (
	orderIdStore = make(map[string]string)
	tempMemory   sync.Mutex
)

func saveOrderId(orderId string, clOrdId string) {
	tempMemory.Lock()
	orderIdStore[clOrdId] = orderId
	tempMemory.Unlock()
}

func getOrderId(clOrdId string) *string {
	tempMemory.Lock()
	defer tempMemory.Unlock()
	orderId, ok := orderIdStore[clOrdId]
	if !ok {
		return nil
	}
	return &orderId
}

func deleteOrderId(clOrdId string) {
	tempMemory.Lock()
	delete(orderIdStore, clOrdId)
	tempMemory.Unlock()
}
