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
		CloseOrderAlgo string `json:"closeOrderAlgo"`
		CTime         string `json:"cTime"`
		UTime         string `json:"uTime"`
		PTime         string `json:"pTime"`
	} `json:"data"`
}
