package main

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/quickfix"
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
	clOrdId := fmt.Sprintf("%d%s", timestamp, randomPart)
	if len(clOrdId) > 36 {
		clOrdId = clOrdId[:36]
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

func convertAsset(asset string) enum.SecurityType {
	switch strings.ToUpper(asset) {
	case "SPOT":
		return enum.SecurityType_FX_SPOT
	case "FUTURE":
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
