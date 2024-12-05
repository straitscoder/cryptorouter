package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/urfave/cli/v2"
)

var newOrderSingleCommand = &cli.Command{
	Name:      "neworder",
	Aliases:   []string{"order"},
	Usage:     "submit order submits an exchange order",
	ArgsUsage: "<exchange> <pair> <side> <type> <amount> <price> <asset> <handle>",
	Action:    NewOrderSingle,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to submit the order for",
		},
		&cli.StringFlag{
			Name:  "pair",
			Usage: "the currency pair (BTC-USDT)",
		},
		&cli.StringFlag{
			Name:  "side",
			Usage: "the order side to use (BUY OR SELL)",
		},
		&cli.StringFlag{
			Name:  "type",
			Usage: "the order type (MARKET OR LIMIT)",
		},
		&cli.Float64Flag{
			Name:  "amount",
			Usage: "the amount for the order",
		},
		&cli.Float64Flag{
			Name:  "price",
			Usage: "the price for the order",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "required asset type",
		},
		&cli.StringFlag{
			Name:  "handle",
			Usage: "how to handle order request (Auto, Semi, or Manual)",
		},
	},
}

func NewOrderSingle(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	var pair string
	if c.IsSet("pair") {
		pair = c.String("pair")
	} else {
		pair = c.Args().Get(1)
	}

	var side string
	if c.IsSet("side") {
		side = c.String("side")
	} else {
		side = c.Args().Get(2)
	}

	var orderType string
	if c.IsSet("type") {
		orderType = c.String("type")
	} else {
		orderType = c.Args().Get(3)
	}

	var amount float64
	if c.IsSet("amount") {
		amount = c.Float64("amount")
	} else {
		var err error
		amount, err = strconv.ParseFloat(c.Args().Get(4), 64)
		if err != nil {
			return err
		}
	}

	var price float64
	if c.IsSet("price") {
		price = c.Float64("price")
	} else {
		var err error
		price, err = strconv.ParseFloat(c.Args().Get(5), 64)
		if err != nil {
			return err
		}
	}

	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(6)
	}

	var handleIns string
	if c.IsSet("handle") {
		handleIns = c.String("handle")
	} else {
		handleIns = c.Args().Get(7)
	}

	if amount == 0 || price == 0 {
		return errors.New("amount and price must be greater than 0")
	}

	req := &NewOrderRequest{
		CliOrdID:  fmt.Sprint(int(time.Now().Unix())),
		Exchange:  exchangeName,
		Pair:      pair,
		Side:      side,
		Type:      orderType,
		Amount:    amount,
		Price:     price,
		Asset:     assetType,
		HandleIns: handleIns,
	}
	jsonOutput(req)

	engine := FixEngine{}

	msg, err := engine.NewOrder(req)
	if err != nil {
		return errors.New(err.Error())
	}
	jsonOutput(msg)
	return nil
}
