package okx

import "context"

// Interface defines the contract for exchange API operations.
// This interface abstracts the underlying exchange client implementation, enabling:
// - Easy testing through mock implementations
// - Flexibility to support multiple exchanges (OKX, Binance, etc.)
// - Adherence to Dependency Inversion Principle (DIP)
//
// All implementations must handle authentication, retry logic, and error handling.
// All methods accept context.Context for cancellation, timeout, and request tracing.
type Interface interface {
	// GetAccountBalance retrieves account balance information from the exchange.
	// Returns balance, available, and frozen amounts for all currencies.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//
	// Returns error on API request failure, authentication failure, or response parsing failure.
	GetAccountBalance(ctx context.Context) (*AccountBalanceResponse, error)

	// GetPositions retrieves all open position information from the exchange.
	// Returns position details including instrument, position size, unrealized PnL, etc.
	// Returns empty data array if no positions exist.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//
	// Returns error on API request failure, authentication failure, or response parsing failure.
	GetPositions(ctx context.Context) (*PositionsResponse, error)

	// GetTicker retrieves market ticker data for a specified instrument.
	// Returns current market price, bid/ask prices, volume, etc.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - instId: Instrument ID (e.g., "BTC-USDT-SWAP")
	//
	// Returns error on API request failure or invalid instrument.
	GetTicker(ctx context.Context, instId string) (*TickerResponse, error)

	// GetPendingAlgoOrders retrieves pending algorithmic orders from the exchange.
	// Returns orders that are waiting to be triggered (e.g., TPSL, conditional orders).
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - ordType: Order type (e.g., "conditional" for TPSL orders)
	//
	// Returns error on API request failure or response parsing failure.
	GetPendingAlgoOrders(ctx context.Context, ordType string) (*PendingAlgoOrdersResponse, error)

	// PlaceAlgoOrder places an algorithmic order on the exchange.
	// Used for placing TPSL orders, conditional orders, etc.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - req: Algo order request with order parameters (instrument, order type, prices, etc.)
	//
	// Returns the order response with algoId (order ID) on success.
	// Returns error on API request failure, authentication failure, or invalid parameters.
	PlaceAlgoOrder(ctx context.Context, req AlgoOrderRequest) (*AlgoOrderResponse, error)

	// PlaceOrder places a regular order on the exchange.
	// Used for opening new positions or closing existing positions.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - req: Order request with order parameters (instrument, side, type, size, price, etc.)
	//
	// Returns the order response with ordId (order ID) on success.
	// Returns error on API request failure, authentication failure, or invalid parameters.
	PlaceOrder(ctx context.Context, req *OrderRequest) (*OrderResponse, error)

	// AmendOrder modifies an existing pending order.
	// Can change order size and/or price for pending orders.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - req: Amend order request with orderId and new parameters
	//
	// Returns the amended order response.
	// Returns error on API request failure or if order cannot be amended.
	AmendOrder(ctx context.Context, req *AmendOrderRequest) (*AmendOrderResponse, error)

	// CancelOrder cancels an existing pending order.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - req: Cancel order request with orderId
	//
	// Returns the cancellation response.
	// Returns error on API request failure or if order cannot be canceled.
	CancelOrder(ctx context.Context, req *CancelOrderRequest) (*CancelOrderResponse, error)

	// GetPendingOrders retrieves all pending orders (not algo orders).
	// Returns orders that are placed but not yet filled or canceled.
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//
	// Returns error on API request failure or response parsing failure.
	GetPendingOrders(ctx context.Context) (*PendingOrdersResponse, error)

	// GetOrdersHistory retrieves order history (completed orders).
	// Returns orders from the last 7 days, including filled and canceled orders.
	// This method fetches ALL orders regardless of how they were placed (app, API, etc).
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//   - instType: Instrument type (SPOT, MARGIN, SWAP, FUTURES, OPTION)
	//   - limit: Number of results (max 100, default 100)
	//
	// Returns error on API request failure or response parsing failure.
	GetOrdersHistory(ctx context.Context, instType string, limit int) (*OrderHistoryResponse, error)

	// HealthCheck verifies exchange API connectivity and authentication.
	// Typically implemented by attempting a simple API call (e.g., GetAccountBalance).
	//
	// Parameters:
	//   - ctx: Context for cancellation and timeout
	//
	// Returns nil if API is reachable and authenticated.
	// Returns error on connection failure or authentication failure.
	HealthCheck(ctx context.Context) error
}

// Ensure Client implements Interface at compile time
var _ Interface = (*Client)(nil)
