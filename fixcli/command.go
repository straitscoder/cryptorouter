package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	"github.com/quickfixgo/fix42/newordersingle"
	"github.com/quickfixgo/quickfix"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

func Fili() error {
	fili := &cobra.Command{
		Use:   "fili",
		Short: "Command line interface for interact with FIX API",
		RunE: func(cmd *cobra.Command, args []string) error {
			fixEngine := new(FixEngine)
			_, cancel := context.WithCancel(context.TODO())
			fixEngine.Start()
		Loop:
			for {
				action, err := Menu()
				if err != nil {
					log.Println(err)
					break
				}

				switch action {
				case "1":
					if err := fixEngine.NewOrder(); err != nil {
						log.Println(err)
						break
					}
					continue Loop
				case "2":
					if err := fixEngine.CancelOrder(); err != nil {
						log.Println(err)
						break
					}
					continue Loop
				case "0":
					break Loop
				default:
					continue Loop
				}
			}

			closeConn(fixEngine.initiator, cancel)
			return cmd.Usage()
		},
	}

	return fili.Execute()
}

func Menu() (string, error) {
	fmt.Println("Menu")
	fmt.Println("1. Order Single")
	fmt.Println("2. Cancel Order")
	fmt.Println("0. Exit")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text(), scanner.Err()
}

func stringField(fieldName string) string {
	fmt.Printf("Please input %s: ", fieldName)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		log.Fatalf("Error reading %s input: %+v", fieldName, err)
	}
	return scanner.Text()
}

func decimalField(fieldName string) decimal.Decimal {
	val, err := decimal.NewFromString(stringField(fieldName))
	if err != nil {
		log.Fatalf("Error reading %s input: %+v", fieldName, err)
	}

	return val
}

func HandleIns() enum.HandlInst {
	fmt.Println("You can choose either Auto, Semi, or Manual")
	handleInsStr := stringField("Handle Instruction")
	return convertHandleInst(strings.ToUpper(handleInsStr))
}

func Symbol() string {
	fmt.Println("Please input symbol you want to trade")
	return stringField("Symbol")
}

func Side() enum.Side {
	fmt.Println("You can choose either Buy or Sell")
	sideStr := stringField("Side")
	return convertSide(strings.ToUpper(sideStr))
}

func Price() decimal.Decimal {
	return decimalField("Price")
}

func Amount() decimal.Decimal {
	return decimalField("Amount")
}

func OrderType() enum.OrdType {
	fmt.Println("You can choose either Limit or Market")
	ordTypeStr := stringField("Order Type")
	return convertOrdType(ordTypeStr)
}

func AssetType() enum.SecurityType {
	fmt.Println("You can choose either Future or Spot")
	assetTypeStr := stringField("Asset Type")
	return convertAsset(assetTypeStr)
}

func Exchange() string {
	return stringField("Exchange")
}

func ClOrdID() string {
	return stringField("Client Order ID")
}

func NewOrderSingle() (msg *quickfix.Message) {
	order := newordersingle.New(
		field.NewClOrdID(string(time.Now().Unix())),
		field.NewHandlInst(HandleIns()),
		field.NewSymbol(Symbol()),
		field.NewSide(Side()),
		field.NewTransactTime(time.Now().UTC()),
		field.NewOrdType(OrderType()),
	)

	order.Set(field.NewPrice(Price(), 8))
	order.Set(field.NewOrderQty(Amount(), 8))

	return order.ToMessage()
}
