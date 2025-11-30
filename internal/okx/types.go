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

// PositionsResponse OKX持仓响应 / OKX positions response
type PositionsResponse struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		InstType      string `json:"instType"`
		MgnMode       string `json:"mgnMode"`
		PosId         string `json:"posId"`
		PosSide       string `json:"posSide"`
		Pos           string `json:"pos"`
		BaseBal       string `json:"baseBal"`
		QuoteBal      string `json:"quoteBal"`
		PosCcy        string `json:"posCcy"`
		AvailPos      string `json:"availPos"`
		AvgPx         string `json:"avgPx"`
		Upl           string `json:"upl"`
		UplRatio      string `json:"uplRatio"`
		UplLastPx     string `json:"uplLastPx"`
		UplRatioLastPx string `json:"uplRatioLastPx"`
		InstId        string `json:"instId"`
		Lever         string `json:"lever"`
		LiqPx         string `json:"liqPx"`
		MarkPx        string `json:"markPx"`
		Imr           string `json:"imr"`
		Margin        string `json:"margin"`
		MgnRatio      string `json:"mgnRatio"`
		Mmr           string `json:"mmr"`
		Liab          string `json:"liab"`
		LiabCcy       string `json:"liabCcy"`
		Interest      string `json:"interest"`
		TradeId       string `json:"tradeId"`
		OptVal        string `json:"optVal"`
		NotionalUsd   string `json:"notionalUsd"`
		Adl           string `json:"adl"`
		Ccy           string `json:"ccy"`
		Last          string `json:"last"`
		UsdPx         string `json:"usdPx"`
		DeltaBS       string `json:"deltaBS"`
		DeltaPA       string `json:"deltaPA"`
		GammaBS       string `json:"gammaBS"`
		GammaPA       string `json:"gammaPA"`
		ThetaBS       string `json:"thetaBS"`
		ThetaPA       string `json:"thetaPA"`
		VegaBS        string `json:"vegaBS"`
		VegaPA        string `json:"vegaPA"`
		SpotInUseAmt  string `json:"spotInUseAmt"`
		ClSpotInUseAmt string `json:"clSpotInUseAmt"`
		RealizedPnl   string `json:"realizedPnl"`
		Pnl           string `json:"pnl"`
		Fee           string `json:"fee"`
		FundingFee    string `json:"fundingFee"`
		LiqPenalty    string `json:"liqPenalty"`
		CloseOrderAlgo []CloseOrderAlgoItem `json:"closeOrderAlgo"`
		CTime         string `json:"cTime"`
		UTime         string `json:"uTime"`
		PTime         string `json:"pTime"`
	} `json:"data"`
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
