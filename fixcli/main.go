package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/signaler"
	"github.com/urfave/cli/v2"
)

func main() {
	app := cli.NewApp()
	app.Name = "fili"
	app.Version = core.Version(true)
	app.EnableBashCompletion = true
	app.Usage = "command line interface for interact with FIX API"
	app.Commands = []*cli.Command{
		newOrderSingleCommand,
	}

	fixEngine := new(FixEngine)
	if err := fixEngine.Start(); err != nil {
		log.Fatal(err)
	}

	context, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		// Capture cancel for interrupt
		signaler.WaitForInterrupt()
		closeConn(fixEngine.initiator, cancel)
		cancel()
		fmt.Println("fix client got interrupted")
		os.Exit(1)
	}()

	if err := app.RunContext(context, os.Args); err != nil {
		log.Fatal(err)
	}
}
