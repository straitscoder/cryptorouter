package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gofrs/uuid"
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
func generateClientOrderID() int64 {
	uuid, _ := uuid.NewV4()
	return int64(uuid[0])
}

func jsonOutput(in interface{}) {
	j, err := json.MarshalIndent(in, "", " ")
	if err != nil {
		return
	}
	fmt.Print(string(j))
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
	case "FUTURES":
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
