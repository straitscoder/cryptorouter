package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	"github.com/quickfixgo/fix42/newordersingle"
	"github.com/quickfixgo/fix42/securitydefinitionrequest"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
	"gopkg.in/ini.v1"
)

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
		parsed := parseFIXMessage(msg)
		jsonOutput(parsed)
	case "8":
		clOrdID, _ := msg.Body.GetString(tag.ClOrdID)
		orderId, _ := msg.Body.GetString(tag.OrderID)
		ordStatus, _ := msg.Body.GetString(tag.OrdStatus)
		savedOrderId := getOrderId(clOrdID)
		if ordStatus == "4" || ordStatus == "3" {
			parsed := parseFIXMessage(msg)
			jsonOutput(parsed)
		}
		if savedOrderId != nil {
			if ordStatus != "0" {
				parsed := parseFIXMessage(msg)
				jsonOutput(parsed)
			}
			return nil
		} else {
			saveOrderId(orderId, clOrdID)
			if ordStatus == "0" {
				parsed := parseFIXMessage(msg)
				jsonOutput(parsed)
				orderResponse := make(map[string]string)
				orderResponse["Client_Order_ID"] = clOrdID
				orderResponse["Order_ID"] = orderId
				jsonOutput(orderResponse)
			}
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

type FixEngine struct {
	senderCompId string
	targetCompId string
	accountCode  string
	initiator    *quickfix.Initiator
	settings     *quickfix.Settings
	logFactory   *quickfix.LogFactory
	storeFactory quickfix.MessageStoreFactory
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

	fe.storeFactory = quickfix.NewMemoryStoreFactory()

	app := &fixApplication{}
	app.Username = config.Section("SESSION").Key("UserName").String()
	app.Password = config.Section("SESSION").Key("Password").String()
	initiator, err := quickfix.NewInitiator(app, fe.storeFactory, fe.settings, *fe.logFactory)
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
		field.NewSecurityReqID(SecReqId()),
		field.NewSecurityRequestType(enum.SecurityRequestType_REQUEST_LIST_SECURITIES),
	)
	securityDefinitionRequest.Set(field.NewSecurityExchange("CCX"))
	sdrMsg := securityDefinitionRequest.ToMessage()
	sdrMsg.Header.Set(field.NewSenderCompID(fe.senderCompId))
	sdrMsg.Header.Set(field.NewTargetCompID(fe.targetCompId))
	parsed := parseFIXMessage(sdrMsg)
	jsonOutput(parsed)
	if !Confirmation() {
		fmt.Println("abort security definition request")
		return nil
	}

	return quickfix.Send(sdrMsg)
}

func (fe *FixEngine) NewOrderSingle() error {
	newOrder := newordersingle.New(
		field.NewClOrdID(generateClOrdID()),
		field.NewHandlInst(HandleIns()),
		field.NewSymbol(Symbol()),
		field.NewSide(Side()),
		field.NewTransactTime(time.Now().UTC()),
		field.NewOrdType(OrderType()),
	)

	newOrder.Set(field.NewAccount(fe.accountCode))
	newOrder.Set(field.NewSecurityType(AssetType()))
	newOrder.Set(field.NewSecurityExchange(Exchange()))
	newOrder.Set(field.NewTimeInForce(TimeInForce()))
	newOrder.Set(field.NewPrice(Price(), 8))
	newOrder.Set(field.NewOrderQty(Amount(), 8))
	orderMsg := newOrder.ToMessage()
	orderMsg.Header.Set(field.NewSenderCompID(fe.senderCompId))
	orderMsg.Header.Set(field.NewTargetCompID(fe.targetCompId))
	parsed := parseFIXMessage(orderMsg)
	jsonOutput(parsed)
	if !Confirmation() {
		fmt.Println("abort new order")
		return nil
	}

	return quickfix.Send(orderMsg)
}
