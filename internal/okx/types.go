package okx

// AccountBalanceResponse OKX账户余额响应 / OKX account balance response
type AccountBalanceResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		TotalEq     string `json:"totalEq"`
		IsoEq       string `json:"isoEq"`
		AdjEq       string `json:"adjEq"`
		OrdFroz     string `json:"ordFroz"`
		Imr         string `json:"imr"`
		Mmr         string `json:"mmr"`
		MgnRatio    string `json:"mgnRatio"`
		NotionalUsd string `json:"notionalUsd"`
		UTime       string `json:"uTime"`
		Details     []struct {
			Ccy           string `json:"ccy"`
			Eq            string `json:"eq"`
			CashBal       string `json:"cashBal"`
			AvailBal      string `json:"availBal"`
			FrozenBal     string `json:"frozenBal"`
			OrdFrozen     string `json:"ordFrozen"`
			Liab          string `json:"liab"`
			Upl           string `json:"upl"`
			UplLib        string `json:"uplLib"`
			CrossLiab     string `json:"crossLiab"`
			IsoLiab       string `json:"isoLiab"`
			MgnRatio      string `json:"mgnRatio"`
			Interest      string `json:"interest"`
			Twap          string `json:"twap"`
			MaxLoan       string `json:"maxLoan"`
			EqUsd         string `json:"eqUsd"`
			NotionalLever string `json:"notionalLever"`
			StgyEq        string `json:"stgyEq"`
			IsoUpl        string `json:"isoUpl"`
			SpotInUseAmt  string `json:"spotInUseAmt"`
		} `json:"details"`
	} `json:"data"`
}

// PositionData OKX持仓数据项 / OKX position data item
type PositionData struct {
	InstType       string               `json:"instType"`
	MgnMode        string               `json:"mgnMode"`
	PosId          string               `json:"posId"`
	PosSide        string               `json:"posSide"`
	Pos            string               `json:"pos"`
	BaseBal        string               `json:"baseBal"`
	QuoteBal       string               `json:"quoteBal"`
	PosCcy         string               `json:"posCcy"`
	AvailPos       string               `json:"availPos"`
	AvgPx          string               `json:"avgPx"`
	Upl            string               `json:"upl"`
	UplRatio       string               `json:"uplRatio"`
	UplLastPx      string               `json:"uplLastPx"`
	UplRatioLastPx string               `json:"uplRatioLastPx"`
	InstId         string               `json:"instId"`
	Lever          string               `json:"lever"`
	LiqPx          string               `json:"liqPx"`
	MarkPx         string               `json:"markPx"`
	Imr            string               `json:"imr"`
	Margin         string               `json:"margin"`
	MgnRatio       string               `json:"mgnRatio"`
	Mmr            string               `json:"mmr"`
	Liab           string               `json:"liab"`
	LiabCcy        string               `json:"liabCcy"`
	Interest       string               `json:"interest"`
	TradeId        string               `json:"tradeId"`
	OptVal         string               `json:"optVal"`
	NotionalUsd    string               `json:"notionalUsd"`
	Adl            string               `json:"adl"`
	Ccy            string               `json:"ccy"`
	Last           string               `json:"last"`
	UsdPx          string               `json:"usdPx"`
	DeltaBS        string               `json:"deltaBS"`
	DeltaPA        string               `json:"deltaPA"`
	GammaBS        string               `json:"gammaBS"`
	GammaPA        string               `json:"gammaPA"`
	ThetaBS        string               `json:"thetaBS"`
	ThetaPA        string               `json:"thetaPA"`
	VegaBS         string               `json:"vegaBS"`
	VegaPA         string               `json:"vegaPA"`
	SpotInUseAmt   string               `json:"spotInUseAmt"`
	ClSpotInUseAmt string               `json:"clSpotInUseAmt"`
	RealizedPnl    string               `json:"realizedPnl"`
	Pnl            string               `json:"pnl"`
	Fee            string               `json:"fee"`
	FundingFee     string               `json:"fundingFee"`
	LiqPenalty     string               `json:"liqPenalty"`
	CloseOrderAlgo []CloseOrderAlgoItem `json:"closeOrderAlgo"`
	CTime          string               `json:"cTime"`
	UTime          string               `json:"uTime"`
	PTime          string               `json:"pTime"`
}

// PositionsResponse OKX持仓响应 / OKX positions response
type PositionsResponse struct {
	Code string         `json:"code"`
	Msg  string         `json:"msg"`
	Data []PositionData `json:"data"`
}

// CloseOrderAlgoItem 持仓关联的止盈止损订单 / Close order algo item attached to position
type CloseOrderAlgoItem struct {
	AlgoId          string `json:"algoId"`
	SlTriggerPx     string `json:"slTriggerPx"`
	SlTriggerPxType string `json:"slTriggerPxType"`
	TpTriggerPx     string `json:"tpTriggerPx"`
	TpTriggerPxType string `json:"tpTriggerPxType"`
	CloseFraction   string `json:"closeFraction"`
}

// AlgoOrderRequest OKX算法订单请求 / OKX algo order request
type AlgoOrderRequest struct {
	InstId        string `json:"instId"`
	TdMode        string `json:"tdMode"`
	Side          string `json:"side"`
	PosSide       string `json:"posSide,omitempty"`
	OrdType       string `json:"ordType"`
	Sz            string `json:"sz"`
	TpTriggerPx   string `json:"tpTriggerPx,omitempty"`
	TpOrdPx       string `json:"tpOrdPx,omitempty"`
	SlTriggerPx   string `json:"slTriggerPx,omitempty"`
	SlOrdPx       string `json:"slOrdPx,omitempty"`
	ReduceOnly    bool   `json:"reduceOnly,omitempty"`
	TpTriggerPxType string `json:"tpTriggerPxType,omitempty"`
	SlTriggerPxType string `json:"slTriggerPxType,omitempty"`
}

// AlgoOrderResponse OKX算法订单响应 / OKX algo order response
type AlgoOrderResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		AlgoId    string `json:"algoId"`
		SCode     string `json:"sCode"`
		SMsg      string `json:"sMsg"`
	} `json:"data"`
}

// PendingAlgoOrdersResponse OKX待处理算法订单响应 / OKX pending algo orders response
type PendingAlgoOrdersResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []AlgoOrder `json:"data"`
}

// AlgoOrder OKX算法订单数据 / OKX algo order data
type AlgoOrder struct {
	AlgoId          string `json:"algoId"`
	InstId          string `json:"instId"`
	PosSide         string `json:"posSide"`
	Side            string `json:"side"`
	Sz              string `json:"sz"`
	OrdType         string `json:"ordType"`
	State           string `json:"state"`
	TpTriggerPx     string `json:"tpTriggerPx"`
	SlTriggerPx     string `json:"slTriggerPx"`
	TpOrdPx         string `json:"tpOrdPx"`
	SlOrdPx         string `json:"slOrdPx"`
	TpTriggerPxType string `json:"tpTriggerPxType"`
	SlTriggerPxType string `json:"slTriggerPxType"`
	ActualSz        string `json:"actualSz"`
	ActualPx        string `json:"actualPx"`
	ActualSide      string `json:"actualSide"`
	CTime           string `json:"cTime"`
	TriggerTime     string `json:"triggerTime"`
	ReduceOnly      string `json:"reduceOnly"`
}

// TickerResponse OKX行情响应 / OKX ticker response
type TickerResponse struct {
	Code string       `json:"code"`
	Msg  string       `json:"msg"`
	Data []TickerData `json:"data"`
}

// TickerData OKX行情数据 / OKX ticker data
type TickerData struct {
	InstId    string `json:"instId"`
	Last      string `json:"last"`      // Last traded price
	LastSz    string `json:"lastSz"`    // Last traded size
	AskPx     string `json:"askPx"`     // Best ask price
	AskSz     string `json:"askSz"`     // Best ask size
	BidPx     string `json:"bidPx"`     // Best bid price
	BidSz     string `json:"bidSz"`     // Best bid size
	Open24h   string `json:"open24h"`   // Open price in the past 24 hours
	High24h   string `json:"high24h"`   // Highest price in the past 24 hours
	Low24h    string `json:"low24h"`    // Lowest price in the past 24 hours
	VolCcy24h string `json:"volCcy24h"` // 24h trading volume (quote currency)
	Vol24h    string `json:"vol24h"`    // 24h trading volume (base currency)
	Ts        string `json:"ts"`        // Ticker data generation time
	SodUtc0   string `json:"sodUtc0"`   // Open price at UTC 0
	SodUtc8   string `json:"sodUtc8"`   // Open price at UTC 8
}

// OrderRequest OKX订单请求 / OKX order request (for regular orders)
type OrderRequest struct {
	InstId     string `json:"instId"`               // Instrument ID
	TdMode     string `json:"tdMode"`               // Trade mode: cash, cross, isolated
	Side       string `json:"side"`                 // Order side: buy, sell
	OrdType    string `json:"ordType"`              // Order type: market, limit, post_only, fok, ioc
	Sz         string `json:"sz"`                   // Order size
	Px         string `json:"px,omitempty"`         // Order price (required for limit orders)
	PosSide    string `json:"posSide,omitempty"`    // Position side: long, short, net
	ReduceOnly bool   `json:"reduceOnly,omitempty"` // Whether the order reduces position only
	TgtCcy     string `json:"tgtCcy,omitempty"`     // Order quantity unit: base_ccy, quote_ccy
}

// OrderResponse OKX订单响应 / OKX order response
type OrderResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		OrdId   string `json:"ordId"`   // Order ID
		ClOrdId string `json:"clOrdId"` // Client order ID
		Tag     string `json:"tag"`     // Order tag
		SCode   string `json:"sCode"`   // Order-specific error code
		SMsg    string `json:"sMsg"`    // Order-specific error message
	} `json:"data"`
}

// AmendOrderRequest OKX修改订单请求 / OKX amend order request
type AmendOrderRequest struct {
	InstId    string `json:"instId"`              // Instrument ID
	OrdId     string `json:"ordId,omitempty"`     // Order ID
	ClOrdId   string `json:"clOrdId,omitempty"`   // Client order ID
	NewSz     string `json:"newSz,omitempty"`     // New order size
	NewPx     string `json:"newPx,omitempty"`     // New order price
	CxlOnFail bool   `json:"cxlOnFail,omitempty"` // Whether to cancel on amendment failure
}

// AmendOrderResponse OKX修改订单响应 / OKX amend order response
type AmendOrderResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		OrdId   string `json:"ordId"`   // Order ID
		ClOrdId string `json:"clOrdId"` // Client order ID
		ReqId   string `json:"reqId"`   // Request ID
		SCode   string `json:"sCode"`   // Order-specific error code
		SMsg    string `json:"sMsg"`    // Order-specific error message
	} `json:"data"`
}

// CancelOrderRequest OKX撤销订单请求 / OKX cancel order request
type CancelOrderRequest struct {
	InstId  string `json:"instId"`            // Instrument ID
	OrdId   string `json:"ordId,omitempty"`   // Order ID
	ClOrdId string `json:"clOrdId,omitempty"` // Client order ID
}

// CancelOrderResponse OKX撤销订单响应 / OKX cancel order response
type CancelOrderResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		OrdId   string `json:"ordId"`   // Order ID
		ClOrdId string `json:"clOrdId"` // Client order ID
		SCode   string `json:"sCode"`   // Order-specific error code
		SMsg    string `json:"sMsg"`    // Order-specific error message
	} `json:"data"`
}

// PendingOrdersResponse OKX待处理订单响应 / OKX pending orders response
type PendingOrdersResponse struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data []OrderData `json:"data"`
}

// OrderData OKX订单数据 / OKX order data
type OrderData struct {
	InstId      string `json:"instId"`      // Instrument ID
	OrdId       string `json:"ordId"`       // Order ID
	ClOrdId     string `json:"clOrdId"`     // Client order ID
	Tag         string `json:"tag"`         // Order tag
	Px          string `json:"px"`          // Order price
	Sz          string `json:"sz"`          // Order size
	OrdType     string `json:"ordType"`     // Order type
	Side        string `json:"side"`        // Order side
	PosSide     string `json:"posSide"`     // Position side
	TdMode      string `json:"tdMode"`      // Trade mode
	AccFillSz   string `json:"accFillSz"`   // Accumulated fill size
	FillPx      string `json:"fillPx"`      // Last fill price
	TradeId     string `json:"tradeId"`     // Last trade ID
	FillSz      string `json:"fillSz"`      // Last fill size
	FillTime    string `json:"fillTime"`    // Last fill time
	State       string `json:"state"`       // Order state: live, partially_filled, etc.
	AvgPx       string `json:"avgPx"`       // Average fill price
	Lever       string `json:"lever"`       // Leverage
	TpTriggerPx string `json:"tpTriggerPx"` // TP trigger price
	TpOrdPx     string `json:"tpOrdPx"`     // TP order price
	SlTriggerPx string `json:"slTriggerPx"` // SL trigger price
	SlOrdPx     string `json:"slOrdPx"`     // SL order price
	Fee         string `json:"fee"`         // Fee
	FeeCcy      string `json:"feeCcy"`      // Fee currency
	Rebate      string `json:"rebate"`      // Rebate
	RebateCcy   string `json:"rebateCcy"`   // Rebate currency
	ReduceOnly  string `json:"reduceOnly"`  // Whether reduce-only order
	CTime       string `json:"cTime"`       // Creation time
	UTime       string `json:"uTime"`       // Update time
}

// OrderHistoryResponse OKX订单历史响应 / OKX order history response
// Used for both 7-day and 3-month order history endpoints
type OrderHistoryResponse struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data []OrderData `json:"data"`
}

// PositionHistoryData OKX历史仓位数据 / OKX position history data
type PositionHistoryData struct {
	InstType      string `json:"instType"`      // Instrument type
	InstId        string `json:"instId"`        // Instrument ID
	MgnMode       string `json:"mgnMode"`       // Margin mode: cross, isolated
	Type          string `json:"type"`          // Close type: 1=partial, 2=all, 3=liquidation, 4=partial_liq, 5=adl
	CTime         string `json:"cTime"`         // Position creation time (ms)
	UTime         string `json:"uTime"`         // Position update time (ms)
	OpenAvgPx     string `json:"openAvgPx"`     // Average opening price
	CloseAvgPx    string `json:"closeAvgPx"`    // Average closing price
	PosId         string `json:"posId"`         // Position ID
	OpenMaxPos    string `json:"openMaxPos"`    // Max position size
	CloseTotalPos string `json:"closeTotalPos"` // Total closed position size
	RealizedPnl   string `json:"realizedPnl"`   // Realized PnL
	Fee           string `json:"fee"`           // Accumulated fee
	FundingFee    string `json:"fundingFee"`    // Accumulated funding fee
	LiqPenalty    string `json:"liqPenalty"`    // Liquidation penalty
	Pnl           string `json:"pnl"`           // PnL (excluding fee)
	PnlRatio      string `json:"pnlRatio"`      // PnL ratio
	PosSide       string `json:"posSide"`       // Position side: long, short, net
	Lever         string `json:"lever"`         // Leverage
	Direction     string `json:"direction"`     // Direction: long, short
	TriggerPx     string `json:"triggerPx"`     // Trigger mark price (for liquidation)
	Uly           string `json:"uly"`           // Underlying
	Ccy           string `json:"ccy"`           // Margin currency
}

// PositionsHistoryResponse OKX历史仓位响应 / OKX positions history response
type PositionsHistoryResponse struct {
	Code string                `json:"code"`
	Msg  string                `json:"msg"`
	Data []PositionHistoryData `json:"data"`
}
