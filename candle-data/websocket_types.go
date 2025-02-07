package main

type OrderTradeUpdate struct {
	EventType       string               `json:"e"`
	EventTime       int64                `json:"E"`
	TransactionTime int64                `json:"T"`
	Data            OrderTradeUpdateData `json:"o"`
}

type OrderTradeUpdateData struct {
	Symbol                   string  `json:"s"`
	ClientOrderID            string  `json:"c"`
	Side                     string  `json:"S"`
	OrderType                string  `json:"o"`
	TimeInForce              string  `json:"f"`
	Quantity                 float64 `json:"q,string"`
	Price                    float64 `json:"p,string"`
	AveragePrice             float64 `json:"ap,string"`
	StopPrice                float64 `json:"sp,string"`
	ExecutionType            string  `json:"x"`
	OrderStatus              string  `json:"X"`
	OrderID                  int64   `json:"i"`
	LastExecutedQuantity     float64 `json:"l,string"`
	CumulativeFilledQuantity float64 `json:"z,string"`
	LastExecutedPrice        float64 `json:"L,string"`
	Commission               float64 `json:"n,string"`
	CommissionAsset          string  `json:"N"`
	TransactionTime          int64   `json:"T"`
	TradeID                  int64   `json:"t"`
	BidsNotional             float64 `json:"b,string"`
	AsksNotional             float64 `json:"a,string"`
	IsMaker                  bool    `json:"m"`
	IsReduceOnly             bool    `json:"R"`
	StopPriceWorkingType     string  `json:"wt"`
	OriginalOrderType        string  `json:"ot"`
	PositionSide             string  `json:"ps"`
	ClosePosition            bool    `json:"cp"`
	ActivationPrice          float64 `json:"AP,string"`
	CallbackRate             float64 `json:"cr,string"`
	PriceProtection          bool    `json:"pP"`
	SI                       int64   `json:"si"`
	SS                       int64   `json:"ss"`
	RealizedProfit           float64 `json:"rp,string"`
	STPMode                  string  `json:"V"`
	PriceMatchMode           string  `json:"pm"`
	TIFGTD                   int64   `json:"gtd"`
}
