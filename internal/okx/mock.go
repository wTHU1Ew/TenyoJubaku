package okx

import (
	"context"
	"fmt"
	"sync"
)

// MockClient is a mock implementation of the OKX Client interface for testing.
// It provides configurable responses for all API methods, enabling comprehensive testing
// without making real API calls.
//
// Example usage:
//
//	mock := &MockClient{
//	    GetPositionsFunc: func() (*PositionsResponse, error) {
//	        return &PositionsResponse{
//	            Code: "0",
//	            Data: []PositionData{
//	                {InstId: "BTC-USDT-SWAP", Pos: "1.5", AvgPx: "50000"},
//	            },
//	        }, nil
//	    },
//	}
//	scheduler := tpsl.NewScheduler(config, mock, logger)
type MockClient struct {
	mu sync.Mutex

	// Counters for tracking call counts
	GetAccountBalanceCallCount   int
	GetPositionsCallCount        int
	GetTickerCallCount           int
	GetPendingAlgoOrdersCallCount int
	PlaceAlgoOrderCallCount      int
	HealthCheckCallCount         int
	PlaceOrderCallCount          int
	AmendOrderCallCount          int
	CancelOrderCallCount         int
	GetPendingOrdersCallCount    int

	// Optional: Configure custom behavior for each method
	GetAccountBalanceFunc   func() (*AccountBalanceResponse, error)
	GetPositionsFunc        func() (*PositionsResponse, error)
	GetTickerFunc           func(instId string) (*TickerResponse, error)
	GetPendingAlgoOrdersFunc func(ordType string) (*PendingAlgoOrdersResponse, error)
	PlaceAlgoOrderFunc      func(req AlgoOrderRequest) (*AlgoOrderResponse, error)
	HealthCheckFunc         func() error

	// Optional: Store captured arguments for assertions
	LastTickerInstId     string
	LastAlgoOrdType      string
	LastAlgoOrderRequest *AlgoOrderRequest
	PlaceOrderArgs       []*OrderRequest

	// Configurable responses for new methods
	PlaceOrderResponse        *OrderResponse
	PlaceOrderError           error
	AmendOrderResponse        *AmendOrderResponse
	AmendOrderError           error
	CancelOrderResponse       *CancelOrderResponse
	CancelOrderError          error
	GetPendingOrdersResponse  *PendingOrdersResponse
	GetPendingOrdersError     error
	TickerResponse            *TickerResponse  // For testing maker-only checks
}

// GetAccountBalance mocks getting account balance.
func (m *MockClient) GetAccountBalance(ctx context.Context) (*AccountBalanceResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetAccountBalanceCallCount++

	// If custom function is provided, use it
	if m.GetAccountBalanceFunc != nil {
		return m.GetAccountBalanceFunc()
	}

	// Default behavior: return empty but valid response
	resp := &AccountBalanceResponse{
		Code: "0",
		Msg:  "",
	}
	resp.Data = make([]struct {
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
	}, 1)

	detail := struct {
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
	}{
		Ccy:       "USDT",
		Eq:        "10000",
		AvailBal:  "10000",
		FrozenBal: "0",
		EqUsd:     "10000",
	}

	resp.Data[0].Details = append(resp.Data[0].Details, detail)
	return resp, nil
}

// GetPositions mocks getting positions.
func (m *MockClient) GetPositions(ctx context.Context) (*PositionsResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetPositionsCallCount++

	// If custom function is provided, use it
	if m.GetPositionsFunc != nil {
		return m.GetPositionsFunc()
	}

	// Default behavior: return empty positions
	return &PositionsResponse{
		Code: "0",
		Msg:  "",
		Data: []PositionData{},
	}, nil
}

// GetTicker mocks getting ticker data.
func (m *MockClient) GetTicker(ctx context.Context, instId string) (*TickerResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetTickerCallCount++
	m.LastTickerInstId = instId

	// If custom function is provided, use it
	if m.GetTickerFunc != nil {
		return m.GetTickerFunc(instId)
	}

	// If TickerResponse is configured, use it
	if m.TickerResponse != nil {
		return m.TickerResponse, nil
	}

	// Default behavior: return mock ticker
	return &TickerResponse{
		Code: "0",
		Msg:  "",
		Data: []TickerData{
			{
				InstId: instId,
				Last:   "50000",
				BidPx:  "49999",
				AskPx:  "50001",
			},
		},
	}, nil
}

// GetPendingAlgoOrders mocks getting pending algo orders.
func (m *MockClient) GetPendingAlgoOrders(ctx context.Context, ordType string) (*PendingAlgoOrdersResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetPendingAlgoOrdersCallCount++
	m.LastAlgoOrdType = ordType

	// If custom function is provided, use it
	if m.GetPendingAlgoOrdersFunc != nil {
		return m.GetPendingAlgoOrdersFunc(ordType)
	}

	// Default behavior: return empty orders
	return &PendingAlgoOrdersResponse{
		Code: "0",
		Msg:  "",
		Data: []AlgoOrder{},
	}, nil
}

// PlaceAlgoOrder mocks placing an algo order.
func (m *MockClient) PlaceAlgoOrder(ctx context.Context, req AlgoOrderRequest) (*AlgoOrderResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PlaceAlgoOrderCallCount++
	m.LastAlgoOrderRequest = &req

	// If custom function is provided, use it
	if m.PlaceAlgoOrderFunc != nil {
		return m.PlaceAlgoOrderFunc(req)
	}

	// Default behavior: return success with mock algoId
	resp := &AlgoOrderResponse{
		Code: "0",
		Msg:  "",
	}
	resp.Data = make([]struct {
		AlgoId string `json:"algoId"`
		SCode  string `json:"sCode"`
		SMsg   string `json:"sMsg"`
	}, 1)
	resp.Data[0].AlgoId = fmt.Sprintf("mock-algo-%d", m.PlaceAlgoOrderCallCount)
	resp.Data[0].SCode = "0"
	resp.Data[0].SMsg = ""
	return resp, nil
}

// HealthCheck mocks health check.
func (m *MockClient) HealthCheck(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.HealthCheckCallCount++

	// If custom function is provided, use it
	if m.HealthCheckFunc != nil {
		return m.HealthCheckFunc()
	}

	// Default behavior: always healthy
	return nil
}

// Reset clears all call counters and captured data (useful for test cleanup).
func (m *MockClient) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetAccountBalanceCallCount = 0
	m.GetPositionsCallCount = 0
	m.GetTickerCallCount = 0
	m.GetPendingAlgoOrdersCallCount = 0
	m.PlaceAlgoOrderCallCount = 0
	m.HealthCheckCallCount = 0

	m.LastTickerInstId = ""
	m.LastAlgoOrdType = ""
	m.LastAlgoOrderRequest = nil
}

// PlaceOrder mock implementation
func (m *MockClient) PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.PlaceOrderCallCount++
	m.PlaceOrderArgs = append(m.PlaceOrderArgs, req)

	if m.PlaceOrderError != nil {
		return nil, m.PlaceOrderError
	}

	return m.PlaceOrderResponse, nil
}

// AmendOrder mock implementation
func (m *MockClient) AmendOrder(ctx context.Context, req *AmendOrderRequest) (*AmendOrderResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.AmendOrderCallCount++

	if m.AmendOrderError != nil {
		return nil, m.AmendOrderError
	}

	return m.AmendOrderResponse, nil
}

// CancelOrder mock implementation
func (m *MockClient) CancelOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CancelOrderCallCount++

	if m.CancelOrderError != nil {
		return nil, m.CancelOrderError
	}

	return m.CancelOrderResponse, nil
}

// GetPendingOrders mock implementation  
func (m *MockClient) GetPendingOrders(ctx context.Context) (*PendingOrdersResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.GetPendingOrdersCallCount++

	if m.GetPendingOrdersError != nil {
		return nil, m.GetPendingOrdersError
	}

	return m.GetPendingOrdersResponse, nil
}
