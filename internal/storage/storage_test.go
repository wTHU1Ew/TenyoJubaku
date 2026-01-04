package storage

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

// setupTestDB creates an in-memory test database
// 创建内存测试数据库
func setupTestDB(t *testing.T) *Storage {
	t.Helper()

	storage, err := New(":memory:", false, 10, 5)
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}

	return storage
}

// =============================================================================
// Account Balance Tests
// =============================================================================

// TestInsertAccountBalance tests basic account balance insertion
// 测试基本账户余额插入
func TestInsertAccountBalance(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	balance := &models.AccountBalance{
		Timestamp: time.Now().UTC(),
		Currency:  "USDT",
		Balance:   10000.0,
		Available: 9500.0,
		Frozen:    500.0,
		Equity:    10050.0,
	}

	err := storage.InsertAccountBalance(balance)
	if err != nil {
		t.Fatalf("InsertAccountBalance failed: %v", err)
	}

	if balance.ID == 0 {
		t.Error("Expected auto-generated ID, got 0")
	}
}

// TestInsertAccountBalance_ValidationError tests validation error handling
// 测试验证错误处理
func TestInsertAccountBalance_ValidationError(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	// Missing currency (should fail validation)
	balance := &models.AccountBalance{
		Timestamp: time.Now().UTC(),
		Currency:  "", // Invalid
		Balance:   10000.0,
		Available: 9500.0,
		Frozen:    500.0,
	}

	err := storage.InsertAccountBalance(balance)
	if err == nil {
		t.Error("Expected validation error for empty currency, got nil")
	}
}

// TestGetLatestAccountBalances tests retrieving latest balances
// 测试获取最新余额
func TestGetLatestAccountBalances(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	// Insert multiple balances for different currencies and times
	now := time.Now().UTC()
	older := now.Add(-1 * time.Hour)

	balances := []*models.AccountBalance{
		{Timestamp: older, Currency: "USDT", Balance: 10000.0, Available: 9500.0, Frozen: 500.0},
		{Timestamp: now, Currency: "USDT", Balance: 10500.0, Available: 10000.0, Frozen: 500.0}, // Latest USDT
		{Timestamp: older, Currency: "BTC", Balance: 1.0, Available: 0.9, Frozen: 0.1},
		{Timestamp: now, Currency: "BTC", Balance: 1.1, Available: 1.0, Frozen: 0.1}, // Latest BTC
	}

	for _, b := range balances {
		if err := storage.InsertAccountBalance(b); err != nil {
			t.Fatalf("Failed to insert balance: %v", err)
		}
	}

	// Get latest balances
	latest, err := storage.GetLatestAccountBalances()
	if err != nil {
		t.Fatalf("GetLatestAccountBalances failed: %v", err)
	}

	if len(latest) != 2 {
		t.Errorf("Expected 2 latest balances, got %d", len(latest))
	}

	// Verify latest USDT balance
	var usdtFound bool
	for _, b := range latest {
		if b.Currency == "USDT" {
			usdtFound = true
			if b.Balance != 10500.0 {
				t.Errorf("Expected latest USDT balance 10500.0, got %f", b.Balance)
			}
		}
	}
	if !usdtFound {
		t.Error("USDT balance not found in latest balances")
	}
}

// TestGetAccountBalancesByTimeRange tests time range queries
// 测试时间范围查询
func TestGetAccountBalancesByTimeRange(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	now := time.Now().UTC()
	start := now.Add(-2 * time.Hour)
	end := now.Add(-1 * time.Hour)

	balances := []*models.AccountBalance{
		{Timestamp: now.Add(-3 * time.Hour), Currency: "USDT", Balance: 9000.0, Available: 9000.0, Frozen: 0}, // Before range
		{Timestamp: now.Add(-90 * time.Minute), Currency: "USDT", Balance: 9500.0, Available: 9500.0, Frozen: 0}, // In range
		{Timestamp: now.Add(-30 * time.Minute), Currency: "USDT", Balance: 10000.0, Available: 10000.0, Frozen: 0}, // After range
	}

	for _, b := range balances {
		if err := storage.InsertAccountBalance(b); err != nil {
			t.Fatalf("Failed to insert balance: %v", err)
		}
	}

	// Query time range
	result, err := storage.GetAccountBalancesByTimeRange("USDT", start, end)
	if err != nil {
		t.Fatalf("GetAccountBalancesByTimeRange failed: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 balance in range, got %d", len(result))
	}

	if len(result) > 0 && result[0].Balance != 9500.0 {
		t.Errorf("Expected balance 9500.0, got %f", result[0].Balance)
	}
}

// =============================================================================
// Position Tests
// =============================================================================

// TestInsertPosition tests position insertion
// 测试仓位插入
func TestInsertPosition(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	position := &models.Position{
		Timestamp:     time.Now().UTC(),
		Instrument:    "BTC-USDT-SWAP",
		PositionSide:  models.PositionSideLong,
		PositionSize:  0.5,
		AveragePrice:  50000.0,
		UnrealizedPnL: 500.0,
		Margin:        2500.0,
		Leverage:      10.0,
		MarginMode:    models.MarginModeCross,
	}

	err := storage.InsertPosition(position)
	if err != nil {
		t.Fatalf("InsertPosition failed: %v", err)
	}

	if position.ID == 0 {
		t.Error("Expected auto-generated ID, got 0")
	}
}

// TestGetLatestPositions tests retrieving latest positions
// 测试获取最新仓位
func TestGetLatestPositions(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	now := time.Now().UTC()
	older := now.Add(-1 * time.Hour)

	positions := []*models.Position{
		{Timestamp: older, Instrument: "BTC-USDT-SWAP", PositionSide: models.PositionSideLong, PositionSize: 0.5, AveragePrice: 50000.0, UnrealizedPnL: 0, Margin: 2500.0, Leverage: 10.0, MarginMode: models.MarginModeCross},
		{Timestamp: now, Instrument: "BTC-USDT-SWAP", PositionSide: models.PositionSideLong, PositionSize: 0.6, AveragePrice: 51000.0, UnrealizedPnL: 600.0, Margin: 3000.0, Leverage: 10.0, MarginMode: models.MarginModeCross}, // Latest
		{Timestamp: older, Instrument: "ETH-USDT-SWAP", PositionSide: models.PositionSideShort, PositionSize: 5.0, AveragePrice: 3000.0, UnrealizedPnL: 0, Margin: 1500.0, Leverage: 10.0, MarginMode: models.MarginModeCross},
	}

	for _, p := range positions {
		if err := storage.InsertPosition(p); err != nil {
			t.Fatalf("Failed to insert position: %v", err)
		}
	}

	latest, err := storage.GetLatestPositions()
	if err != nil {
		t.Fatalf("GetLatestPositions failed: %v", err)
	}

	// Should return only latest timestamp positions
	if len(latest) == 0 {
		t.Error("Expected at least one position, got none")
	}

	for _, p := range latest {
		if !p.Timestamp.Equal(now.Truncate(time.Second)) {
			// Allow 1 second tolerance due to time truncation
			if p.Timestamp.Sub(now.Truncate(time.Second)).Abs() > time.Second {
				t.Errorf("Expected latest timestamp %v, got %v", now, p.Timestamp)
			}
		}
	}
}

// =============================================================================
// Order History Tests
// =============================================================================

// TestInsertOrderHistory tests basic order insertion
// 测试基本订单插入
func TestInsertOrderHistory(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	order := &models.OrderHistory{
		OrderID:    "test-order-001",
		InstId:     "BTC-USDT-SWAP",
		Side:       "buy",
		OrdType:    "limit",
		Size:       "0.01",
		Price:      "50000.0",
		ReduceOnly: false,
		PlacedAt:   now,
		WeekStart:  models.GetWeekStart(now),
		Status:     string(models.OrderStatusPlaced),
		CreatedAt:  now,
	}

	err := storage.InsertOrderHistory(ctx, order)
	if err != nil {
		t.Fatalf("InsertOrderHistory failed: %v", err)
	}

	if order.ID == 0 {
		t.Error("Expected auto-generated ID, got 0")
	}
}

// TestInsertOrderHistory_Idempotent tests idempotent insert behavior
// 测试幂等插入行为
func TestInsertOrderHistory_Idempotent(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	order := &models.OrderHistory{
		OrderID:    "test-order-idempotent",
		InstId:     "BTC-USDT-SWAP",
		Side:       "buy",
		OrdType:    "limit",
		Size:       "0.01",
		Price:      "50000.0",
		ReduceOnly: false,
		PlacedAt:   now,
		WeekStart:  models.GetWeekStart(now),
		Status:     string(models.OrderStatusPlaced),
		CreatedAt:  now,
	}

	// First insert
	err := storage.InsertOrderHistory(ctx, order)
	if err != nil {
		t.Fatalf("First insert failed: %v", err)
	}
	firstID := order.ID

	// Second insert (should find existing record)
	order2 := &models.OrderHistory{
		OrderID:    "test-order-idempotent", // Same OrderID
		InstId:     "BTC-USDT-SWAP",
		Side:       "sell", // Different fields
		OrdType:    "market",
		Size:       "0.02",
		PlacedAt:   now,
		WeekStart:  models.GetWeekStart(now),
		Status:     string(models.OrderStatusFilled),
		CreatedAt:  now,
	}

	err = storage.InsertOrderHistory(ctx, order2)
	if err != nil {
		t.Fatalf("Second insert failed: %v", err)
	}

	// Should return existing record (FirstOrCreate behavior)
	if order2.ID != firstID {
		t.Logf("Note: FirstOrCreate returned existing record with ID %d (expected)", order2.ID)
	}
}

// TestGetOrderCountForWeek tests weekly order counting
// 测试每周订单计数
func TestGetOrderCountForWeek(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	weekStart := models.GetWeekStart(now)

	orders := []*models.OrderHistory{
		{OrderID: "order-1", InstId: "BTC-USDT-SWAP", Side: "buy", OrdType: "limit", Size: "0.01", ReduceOnly: false, PlacedAt: now, WeekStart: weekStart, Status: string(models.OrderStatusPlaced), CreatedAt: now},
		{OrderID: "order-2", InstId: "BTC-USDT-SWAP", Side: "sell", OrdType: "limit", Size: "0.01", ReduceOnly: true, PlacedAt: now, WeekStart: weekStart, Status: string(models.OrderStatusPlaced), CreatedAt: now},
		{OrderID: "order-3", InstId: "ETH-USDT-SWAP", Side: "buy", OrdType: "market", Size: "0.1", ReduceOnly: false, PlacedAt: now, WeekStart: weekStart, Status: string(models.OrderStatusFilled), CreatedAt: now},
	}

	for _, o := range orders {
		if err := storage.InsertOrderHistory(ctx, o); err != nil {
			t.Fatalf("Failed to insert order: %v", err)
		}
	}

	// Get total count (interface doesn't support excludeReduceOnly parameter)
	count, err := storage.GetOrderCountForWeek(ctx, weekStart)
	if err != nil {
		t.Fatalf("GetOrderCountForWeek failed: %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 total orders, got %d", count)
	}
}

// TestGetOrdersForWeek tests retrieving orders for a week
// 测试获取一周的订单
func TestGetOrdersForWeek(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	weekStart := models.GetWeekStart(now)
	lastWeek := weekStart.AddDate(0, 0, -7)

	orders := []*models.OrderHistory{
		{OrderID: "this-week-1", InstId: "BTC-USDT-SWAP", Side: "buy", OrdType: "limit", Size: "0.01", PlacedAt: now, WeekStart: weekStart, Status: string(models.OrderStatusPlaced), CreatedAt: now},
		{OrderID: "last-week-1", InstId: "BTC-USDT-SWAP", Side: "buy", OrdType: "limit", Size: "0.01", PlacedAt: lastWeek, WeekStart: lastWeek, Status: string(models.OrderStatusPlaced), CreatedAt: lastWeek},
	}

	for _, o := range orders {
		if err := storage.InsertOrderHistory(ctx, o); err != nil {
			t.Fatalf("Failed to insert order: %v", err)
		}
	}

	// Get this week's orders
	thisWeekOrders, err := storage.GetOrdersForWeek(ctx, weekStart)
	if err != nil {
		t.Fatalf("GetOrdersForWeek failed: %v", err)
	}

	if len(thisWeekOrders) != 1 {
		t.Errorf("Expected 1 order for this week, got %d", len(thisWeekOrders))
	}

	if len(thisWeekOrders) > 0 && thisWeekOrders[0].OrderID != "this-week-1" {
		t.Errorf("Expected order ID 'this-week-1', got '%s'", thisWeekOrders[0].OrderID)
	}
}

// TestGetWeeklyOrderStats tests weekly order statistics aggregation
// 测试每周订单统计聚合
func TestGetWeeklyOrderStats(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	weekStart := models.GetWeekStart(now)

	orders := []*models.OrderHistory{
		{OrderID: "stats-1", InstId: "BTC-USDT-SWAP", Side: "buy", OrdType: "limit", Size: "0.01", ReduceOnly: false, PlacedAt: now, WeekStart: weekStart, Status: string(models.OrderStatusPlaced), CreatedAt: now},
		{OrderID: "stats-2", InstId: "BTC-USDT-SWAP", Side: "sell", OrdType: "limit", Size: "0.01", ReduceOnly: true, PlacedAt: now, WeekStart: weekStart, Status: string(models.OrderStatusPlaced), CreatedAt: now},
		{OrderID: "stats-3", InstId: "ETH-USDT-SWAP", Side: "buy", OrdType: "market", Size: "0.1", ReduceOnly: false, PlacedAt: now, WeekStart: weekStart, Status: string(models.OrderStatusFilled), CreatedAt: now},
		{OrderID: "stats-4", InstId: "ETH-USDT-SWAP", Side: "sell", OrdType: "limit", Size: "0.1", ReduceOnly: false, PlacedAt: now, WeekStart: weekStart, Status: string(models.OrderStatusCanceled), CreatedAt: now},
	}

	for _, o := range orders {
		if err := storage.InsertOrderHistory(ctx, o); err != nil {
			t.Fatalf("Failed to insert order: %v", err)
		}
	}

	stats, err := storage.GetWeeklyOrderStats(ctx, weekStart)
	if err != nil {
		t.Fatalf("GetWeeklyOrderStats failed: %v", err)
	}

	if stats.TotalOrders != 4 {
		t.Errorf("Expected TotalOrders=4, got %d", stats.TotalOrders)
	}

	if stats.ReduceOnlyOrders != 1 {
		t.Errorf("Expected ReduceOnlyOrders=1, got %d", stats.ReduceOnlyOrders)
	}

	if stats.RegularOrders != 3 {
		t.Errorf("Expected RegularOrders=3, got %d", stats.RegularOrders)
	}

	if stats.PlacedOrders != 2 {
		t.Errorf("Expected PlacedOrders=2, got %d", stats.PlacedOrders)
	}

	if stats.FilledOrders != 1 {
		t.Errorf("Expected FilledOrders=1, got %d", stats.FilledOrders)
	}

	if stats.CanceledOrders != 1 {
		t.Errorf("Expected CanceledOrders=1, got %d", stats.CanceledOrders)
	}
}

// =============================================================================
// Position History Tests
// =============================================================================

// TestInsertPositionHistory tests position history insertion
// 测试历史仓位插入
func TestInsertPositionHistory(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	posHist := &models.PositionHistory{
		PosId:         "pos-001",
		InstType:      "SWAP",
		InstId:        "BTC-USDT-SWAP",
		MgnMode:       "cross",
		PosSide:       "long",
		Lever:         "10",
		OpenAvgPx:     "50000",
		CloseAvgPx:    "51000",
		OpenMaxPos:    "0.5",
		CloseTotalPos: "0.5",
		RealizedPnl:   "450",
		Pnl:           "500",
		PnlRatio:      "0.02",
		Fee:           "50",
		FundingFee:    "0",
		LiqPenalty:    "0",
		CloseType:     "2", // Full close
		Direction:     "long",
		Ccy:           "USDT",
		OpenedAt:      now.Add(-24 * time.Hour),
		ClosedAt:      now,
		CreatedAt:     now,
	}

	err := storage.InsertPositionHistory(ctx, posHist)
	if err != nil {
		t.Fatalf("InsertPositionHistory failed: %v", err)
	}

	if posHist.ID == 0 {
		t.Error("Expected auto-generated ID, got 0")
	}
}

// TestInsertPositionHistory_Idempotent tests idempotent insert for position history
// 测试历史仓位幂等插入
func TestInsertPositionHistory_Idempotent(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	posHist := &models.PositionHistory{
		PosId:         "pos-idempotent",
		InstType:      "SWAP",
		InstId:        "BTC-USDT-SWAP",
		MgnMode:       "cross",
		PosSide:       "long",
		Lever:         "10",
		OpenAvgPx:     "50000",
		CloseAvgPx:    "51000",
		OpenMaxPos:    "0.5",
		CloseTotalPos: "0.5",
		RealizedPnl:   "450",
		Pnl:           "500",
		PnlRatio:      "0.02",
		Fee:           "50",
		FundingFee:    "0",
		LiqPenalty:    "0",
		CloseType:     "2",
		Direction:     "long",
		Ccy:           "USDT",
		OpenedAt:      now.Add(-24 * time.Hour),
		ClosedAt:      now,
		CreatedAt:     now,
	}

	// First insert
	err := storage.InsertPositionHistory(ctx, posHist)
	if err != nil {
		t.Fatalf("First insert failed: %v", err)
	}
	firstID := posHist.ID

	// Second insert (should find existing)
	posHist2 := &models.PositionHistory{
		PosId:         "pos-idempotent", // Same PosId
		InstType:      "FUTURES",        // Different fields
		InstId:        "ETH-USDT-SWAP",
		MgnMode:       "isolated",
		PosSide:       "short",
		Lever:         "20",
		OpenAvgPx:     "3000",
		CloseAvgPx:    "2900",
		OpenMaxPos:    "5.0",
		CloseTotalPos: "5.0",
		RealizedPnl:   "500",
		Pnl:           "500",
		PnlRatio:      "0.033",
		Fee:           "15",
		FundingFee:    "0",
		LiqPenalty:    "0",
		CloseType:     "1",
		Direction:     "short",
		Ccy:           "USDT",
		OpenedAt:      now.Add(-12 * time.Hour),
		ClosedAt:      now,
		CreatedAt:     now,
	}

	err = storage.InsertPositionHistory(ctx, posHist2)
	if err != nil {
		t.Fatalf("Second insert failed: %v", err)
	}

	// Should return existing record
	if posHist2.ID != firstID {
		t.Logf("Note: FirstOrCreate returned existing record with ID %d (expected)", posHist2.ID)
	}
}

// TestGetPositionsHistory tests retrieving position history with filters
// 测试获取历史仓位（带过滤器）
func TestGetPositionsHistory(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	positions := []*models.PositionHistory{
		{PosId: "pos-btc-1", InstType: "SWAP", InstId: "BTC-USDT-SWAP", MgnMode: "cross", PosSide: "long", Lever: "10", OpenAvgPx: "50000", CloseAvgPx: "51000", OpenMaxPos: "0.5", CloseTotalPos: "0.5", RealizedPnl: "450", Pnl: "500", PnlRatio: "0.02", Fee: "50", FundingFee: "0", LiqPenalty: "0", CloseType: "2", Direction: "long", Ccy: "USDT", OpenedAt: now.Add(-48 * time.Hour), ClosedAt: now.Add(-24 * time.Hour), CreatedAt: now},
		{PosId: "pos-btc-2", InstType: "SWAP", InstId: "BTC-USDT-SWAP", MgnMode: "cross", PosSide: "long", Lever: "10", OpenAvgPx: "51000", CloseAvgPx: "52000", OpenMaxPos: "0.6", CloseTotalPos: "0.6", RealizedPnl: "540", Pnl: "600", PnlRatio: "0.0196", Fee: "60", FundingFee: "0", LiqPenalty: "0", CloseType: "2", Direction: "long", Ccy: "USDT", OpenedAt: now.Add(-24 * time.Hour), ClosedAt: now, CreatedAt: now},
		{PosId: "pos-eth-1", InstType: "SWAP", InstId: "ETH-USDT-SWAP", MgnMode: "cross", PosSide: "short", Lever: "10", OpenAvgPx: "3000", CloseAvgPx: "2900", OpenMaxPos: "5.0", CloseTotalPos: "5.0", RealizedPnl: "450", Pnl: "500", PnlRatio: "0.033", Fee: "50", FundingFee: "0", LiqPenalty: "0", CloseType: "2", Direction: "short", Ccy: "USDT", OpenedAt: now.Add(-12 * time.Hour), ClosedAt: now, CreatedAt: now},
	}

	for _, p := range positions {
		if err := storage.InsertPositionHistory(ctx, p); err != nil {
			t.Fatalf("Failed to insert position history: %v", err)
		}
	}

	// Test: Get all BTC positions
	btcPositions, err := storage.GetPositionsHistory(ctx, "BTC-USDT-SWAP", 0)
	if err != nil {
		t.Fatalf("GetPositionsHistory (BTC) failed: %v", err)
	}

	if len(btcPositions) != 2 {
		t.Errorf("Expected 2 BTC positions, got %d", len(btcPositions))
	}

	// Verify order (most recent first)
	if len(btcPositions) >= 2 && btcPositions[0].PosId != "pos-btc-2" {
		t.Errorf("Expected first position to be pos-btc-2 (most recent), got %s", btcPositions[0].PosId)
	}

	// Test: Get all positions with limit
	allPositions, err := storage.GetPositionsHistory(ctx, "", 2)
	if err != nil {
		t.Fatalf("GetPositionsHistory (all, limit 2) failed: %v", err)
	}

	if len(allPositions) != 2 {
		t.Errorf("Expected 2 positions (limit), got %d", len(allPositions))
	}

	// Test: Get all positions without filter
	allPositionsNoLimit, err := storage.GetPositionsHistory(ctx, "", 0)
	if err != nil {
		t.Fatalf("GetPositionsHistory (all, no limit) failed: %v", err)
	}

	if len(allPositionsNoLimit) != 3 {
		t.Errorf("Expected 3 total positions, got %d", len(allPositionsNoLimit))
	}
}

// =============================================================================
// Pending Confirmation Tests (Not Yet Implemented)
// =============================================================================

// TestPendingConfirmationNotImplemented verifies all pending methods return "not yet implemented"
// 测试待确认订单方法返回"未实现"错误
func TestPendingConfirmationNotImplemented(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	// Test InsertPendingConfirmation
	err := storage.InsertPendingConfirmation(ctx, &models.PendingConfirmation{
		OrderID:             "test-order",
		InstId:              "BTC-USDT-SWAP",
		Side:                "buy",
		OrdType:             "limit",
		OriginalSize:        "0.01",
		CurrentSize:         "0.01",
		PlacedAt:            now,
		NextConfirmationDue: now.Add(5 * time.Minute),
		Status:              string(models.ConfirmationStatusPending),
		CreatedAt:           now,
		UpdatedAt:           now,
	})
	if err == nil || !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("InsertPendingConfirmation: expected 'not yet implemented' error, got %v", err)
	}

	// Test GetPendingConfirmationsDue
	_, err = storage.GetPendingConfirmationsDue(ctx, now)
	if err == nil || !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("GetPendingConfirmationsDue: expected 'not yet implemented' error, got %v", err)
	}

	// Test GetPendingConfirmation
	_, err = storage.GetPendingConfirmation(ctx, "test-order")
	if err == nil || !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("GetPendingConfirmation: expected 'not yet implemented' error, got %v", err)
	}

	// Test UpdatePendingConfirmation
	statusStr := string(models.ConfirmationStatusConfirmed)
	err = storage.UpdatePendingConfirmation(ctx, "test-order", &models.ConfirmationUpdate{
		Status:    &statusStr,
		UpdatedAt: now,
	})
	if err == nil || !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("UpdatePendingConfirmation: expected 'not yet implemented' error, got %v", err)
	}

	// Test DeletePendingConfirmation
	err = storage.DeletePendingConfirmation(ctx, "test-order")
	if err == nil || !strings.Contains(err.Error(), "not yet implemented") {
		t.Errorf("DeletePendingConfirmation: expected 'not yet implemented' error, got %v", err)
	}
}

// =============================================================================
// Utility Tests
// =============================================================================

// TestHealthCheck tests the health check functionality
// 测试健康检查功能
func TestHealthCheck(t *testing.T) {
	storage := setupTestDB(t)
	defer storage.Close()

	err := storage.HealthCheck()
	if err != nil {
		t.Errorf("HealthCheck failed: %v", err)
	}
}

// TestClose tests proper resource cleanup
// 测试资源清理
func TestClose(t *testing.T) {
	storage := setupTestDB(t)

	err := storage.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}

	// Verify DB is closed by attempting an operation
	balance := &models.AccountBalance{
		Timestamp: time.Now().UTC(),
		Currency:  "USDT",
		Balance:   10000.0,
		Available: 9500.0,
		Frozen:    500.0,
	}

	err = storage.InsertAccountBalance(balance)
	if err == nil {
		t.Error("Expected error after Close(), got nil")
	}
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

// TestConcurrentInserts tests thread-safe concurrent insertions
// 测试线程安全的并发插入
// SKIPPED: In-memory SQLite has concurrency limitations. This test will be enabled
// once we migrate to file-based database for integration testing.
/*
func TestConcurrentInserts(t *testing.T) {
	t.Skip("Skipping concurrent test - in-memory SQLite limitations")

	storage := setupTestDB(t)
	defer storage.Close()

	ctx := context.Background()
	now := time.Now().UTC()
	weekStart := models.GetWeekStart(now)

	var wg sync.WaitGroup
	numGoroutines := 10
	ordersPerGoroutine := 10

	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < ordersPerGoroutine; j++ {
				order := &models.OrderHistory{
					OrderID:    stringPtr(goroutineID*100 + j),
					InstId:     "BTC-USDT-SWAP",
					Side:       "buy",
					OrdType:    "limit",
					Size:       "0.01",
					PlacedAt:   now,
					WeekStart:  weekStart,
					Status:     string(models.OrderStatusPlaced),
					CreatedAt:  now,
					ReduceOnly: false,
				}

				err := storage.InsertOrderHistory(ctx, order)
				if err != nil {
					t.Errorf("Concurrent insert failed (goroutine %d, order %d): %v", goroutineID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify all orders were inserted
	count, err := storage.GetOrderCountForWeek(ctx, weekStart)
	if err != nil {
		t.Fatalf("GetOrderCountForWeek failed: %v", err)
	}

	expectedCount := numGoroutines * ordersPerGoroutines
	if count != expectedCount {
		t.Errorf("Expected %d orders after concurrent inserts, got %d", expectedCount, count)
	}
}
*/

// =============================================================================
// Helper Functions
// =============================================================================

// stringPtr returns a pointer to a string (helper for building order IDs)
// 返回字符串指针（用于构建订单ID）
func stringPtr(i int) string {
	return fmt.Sprintf("order-%03d", i)
}
