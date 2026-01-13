package tpsl

import (
	"context"
	"testing"
	"time"

	"github.com/wTHU1Ew/TenyoJubaku/internal/config"
	"github.com/wTHU1Ew/TenyoJubaku/internal/storage"
	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

// setupTestStorage 创建测试用的in-memory存储 / Create in-memory storage for testing
func setupTestStorage(t *testing.T) storage.Interface {
	t.Helper()
	db, err := storage.New(":memory:", false, 10, 5)
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}
	return db
}

// TestLoadOrCreateTracker_CreateNew 测试创建新的追踪器 / Test creating new tracker
func TestLoadOrCreateTracker_CreateNew(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	position := &models.Position{
		Instrument:   "BTC-USDT-SWAP",
		PositionSide: models.PositionSideLong,
		AveragePrice: 50000.0,
	}
	currentSlPrice := 49000.0

	tracker, err := LoadOrCreateTracker(ctx, storage, position, currentSlPrice)
	if err != nil {
		t.Fatalf("LoadOrCreateTracker failed: %v", err)
	}

	// Verify tracker fields
	if tracker.PositionKey != "BTC-USDT-SWAP_long" {
		t.Errorf("Expected PositionKey='BTC-USDT-SWAP_long', got '%s'", tracker.PositionKey)
	}
	if tracker.EntryPrice != 50000.0 {
		t.Errorf("Expected EntryPrice=50000.0, got %.2f", tracker.EntryPrice)
	}
	if tracker.CurrentSlPrice != 49000.0 {
		t.Errorf("Expected CurrentSlPrice=49000.0, got %.2f", tracker.CurrentSlPrice)
	}
	if tracker.HighestPriceReached != 50000.0 {
		t.Errorf("Expected HighestPriceReached=50000.0 (long position), got %.2f", tracker.HighestPriceReached)
	}
	if tracker.LowestPriceReached != 0 {
		t.Errorf("Expected LowestPriceReached=0 (long position), got %.2f", tracker.LowestPriceReached)
	}
	if tracker.FirstMoveTriggered {
		t.Error("Expected FirstMoveTriggered=false for new tracker")
	}
}

// TestLoadOrCreateTracker_LoadExisting 测试加载已存在的追踪器 / Test loading existing tracker
func TestLoadOrCreateTracker_LoadExisting(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	position := &models.Position{
		Instrument:   "ETH-USDT-SWAP",
		PositionSide: models.PositionSideLong,
		AveragePrice: 3000.0,
	}
	currentSlPrice := 2950.0

	// Create tracker first time
	tracker1, err := LoadOrCreateTracker(ctx, storage, position, currentSlPrice)
	if err != nil {
		t.Fatalf("First LoadOrCreateTracker failed: %v", err)
	}
	originalID := tracker1.ID

	// Load tracker second time (should return existing)
	tracker2, err := LoadOrCreateTracker(ctx, storage, position, currentSlPrice)
	if err != nil {
		t.Fatalf("Second LoadOrCreateTracker failed: %v", err)
	}

	if tracker2.ID != originalID {
		t.Errorf("Expected same tracker ID=%d, got %d", originalID, tracker2.ID)
	}
}

// TestLoadOrCreateTracker_ShortPosition 测试创建空头持仓追踪器 / Test creating tracker for short position
func TestLoadOrCreateTracker_ShortPosition(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	position := &models.Position{
		Instrument:   "BTC-USDT-SWAP",
		PositionSide: models.PositionSideShort,
		AveragePrice: 50000.0,
	}
	currentSlPrice := 51000.0

	tracker, err := LoadOrCreateTracker(ctx, storage, position, currentSlPrice)
	if err != nil {
		t.Fatalf("LoadOrCreateTracker failed: %v", err)
	}

	// For short positions, track lowest price
	if tracker.LowestPriceReached != 50000.0 {
		t.Errorf("Expected LowestPriceReached=50000.0 (short position), got %.2f", tracker.LowestPriceReached)
	}
	if tracker.HighestPriceReached != 0 {
		t.Errorf("Expected HighestPriceReached=0 (short position), got %.2f", tracker.HighestPriceReached)
	}
}

// TestUpdateTracker_LongHighestPrice 测试更新多头最高价格 / Test updating highest price for long
func TestUpdateTracker_LongHighestPrice(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	tracker := &models.DynamicSLTracker{
		PositionKey:         "BTC-USDT-SWAP_long",
		InstId:              "BTC-USDT-SWAP",
		PosSide:             "long",
		EntryPrice:          50000.0,
		CurrentSlPrice:      49000.0,
		HighestPriceReached: 50000.0,
		LowestPriceReached:  0,
		FirstMoveTriggered:  false,
		CreatedAt:           time.Now().UTC(),
		LastUpdatedAt:       time.Now().UTC(),
	}

	// Insert tracker
	err := storage.InsertDynamicSLTracker(ctx, tracker)
	if err != nil {
		t.Fatalf("Failed to insert tracker: %v", err)
	}

	config := &config.DynamicSLConfig{
		FirstMovePct: 0.01, // 1%
	}

	// Update with higher price
	currentPrice := 50500.0
	updated, err := UpdateTracker(ctx, storage, tracker, currentPrice, config)
	if err != nil {
		t.Fatalf("UpdateTracker failed: %v", err)
	}

	if !updated {
		t.Error("Expected updated=true when price increases")
	}
	if tracker.HighestPriceReached != 50500.0 {
		t.Errorf("Expected HighestPriceReached=50500.0, got %.2f", tracker.HighestPriceReached)
	}
}

// TestUpdateTracker_FirstMoveTriggered 测试firstMove阈值触发 / Test firstMove threshold triggering
func TestUpdateTracker_FirstMoveTriggered(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	tracker := &models.DynamicSLTracker{
		PositionKey:         "BTC-USDT-SWAP_long",
		InstId:              "BTC-USDT-SWAP",
		PosSide:             "long",
		EntryPrice:          50000.0,
		CurrentSlPrice:      49000.0,
		HighestPriceReached: 50000.0,
		LowestPriceReached:  0,
		FirstMoveTriggered:  false,
		CreatedAt:           time.Now().UTC(),
		LastUpdatedAt:       time.Now().UTC(),
	}

	err := storage.InsertDynamicSLTracker(ctx, tracker)
	if err != nil {
		t.Fatalf("Failed to insert tracker: %v", err)
	}

	config := &config.DynamicSLConfig{
		FirstMovePct: 0.01, // 1%
	}

	// Update with price that triggers firstMove (50000 * 1.01 = 50500)
	currentPrice := 50500.0
	updated, err := UpdateTracker(ctx, storage, tracker, currentPrice, config)
	if err != nil {
		t.Fatalf("UpdateTracker failed: %v", err)
	}

	if !updated {
		t.Error("Expected updated=true when firstMove triggered")
	}
	if !tracker.FirstMoveTriggered {
		t.Error("Expected FirstMoveTriggered=true after 1% profit")
	}
}

// TestUpdateTracker_ShortLowestPrice 测试更新空头最低价格 / Test updating lowest price for short
func TestUpdateTracker_ShortLowestPrice(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	tracker := &models.DynamicSLTracker{
		PositionKey:         "BTC-USDT-SWAP_short",
		InstId:              "BTC-USDT-SWAP",
		PosSide:             "short",
		EntryPrice:          50000.0,
		CurrentSlPrice:      51000.0,
		HighestPriceReached: 0,
		LowestPriceReached:  50000.0,
		FirstMoveTriggered:  false,
		CreatedAt:           time.Now().UTC(),
		LastUpdatedAt:       time.Now().UTC(),
	}

	err := storage.InsertDynamicSLTracker(ctx, tracker)
	if err != nil {
		t.Fatalf("Failed to insert tracker: %v", err)
	}

	config := &config.DynamicSLConfig{
		FirstMovePct: 0.01, // 1%
	}

	// Update with lower price (profit for short)
	currentPrice := 49500.0
	updated, err := UpdateTracker(ctx, storage, tracker, currentPrice, config)
	if err != nil {
		t.Fatalf("UpdateTracker failed: %v", err)
	}

	if !updated {
		t.Error("Expected updated=true when price decreases (short)")
	}
	if tracker.LowestPriceReached != 49500.0 {
		t.Errorf("Expected LowestPriceReached=49500.0, got %.2f", tracker.LowestPriceReached)
	}
	if !tracker.FirstMoveTriggered {
		t.Error("Expected FirstMoveTriggered=true after 1% profit for short")
	}
}

// TestCalculateDynamicSL_BeforeFirstMove 测试firstMove前不调整 / Test no adjustment before firstMove
func TestCalculateDynamicSL_BeforeFirstMove(t *testing.T) {
	config := &config.DynamicSLConfig{
		FirstMovePct:     0.01, // 1%
		TrailingStepPct:  0.005, // 0.5%
		StopMoveStepPct:  0.001, // 0.1%
	}

	entryPrice := 50000.0
	currentPrice := 50400.0 // Only 0.8% profit
	highestPrice := 50400.0
	currentSlPrice := 49000.0
	firstMoveTriggered := false
	isLong := true

	shouldAdjust, newSlPrice, err := CalculateDynamicSL(
		entryPrice, currentPrice, highestPrice, currentSlPrice,
		firstMoveTriggered, isLong, config,
	)

	if err != nil {
		t.Fatalf("CalculateDynamicSL failed: %v", err)
	}
	if shouldAdjust {
		t.Error("Expected shouldAdjust=false when profit < firstMovePct")
	}
	if newSlPrice != 0 {
		t.Errorf("Expected newSlPrice=0 when not adjusting, got %.2f", newSlPrice)
	}
}

// TestCalculateDynamicSL_FirstMoveTriggered 测试firstMove触发后移动到盈亏平衡 / Test move to breakeven on firstMove
func TestCalculateDynamicSL_FirstMoveTriggered(t *testing.T) {
	config := &config.DynamicSLConfig{
		FirstMovePct:     0.01, // 1%
		TrailingStepPct:  0.005, // 0.5%
		StopMoveStepPct:  0.001, // 0.1%
	}

	entryPrice := 50000.0
	currentPrice := 50500.0 // 1% profit
	highestPrice := 50500.0
	currentSlPrice := 49000.0
	firstMoveTriggered := false
	isLong := true

	shouldAdjust, newSlPrice, err := CalculateDynamicSL(
		entryPrice, currentPrice, highestPrice, currentSlPrice,
		firstMoveTriggered, isLong, config,
	)

	if err != nil {
		t.Fatalf("CalculateDynamicSL failed: %v", err)
	}
	if !shouldAdjust {
		t.Error("Expected shouldAdjust=true when firstMove triggered")
	}

	expectedBreakeven := entryPrice * 1.001 // 50050
	if newSlPrice < expectedBreakeven-0.1 || newSlPrice > expectedBreakeven+0.1 {
		t.Errorf("Expected newSlPrice≈%.2f (breakeven), got %.2f", expectedBreakeven, newSlPrice)
	}
}

// TestCalculateDynamicSL_MoveToBreakeven 测试移动到盈亏平衡点 / Test ensuring SL at breakeven
func TestCalculateDynamicSL_MoveToBreakeven(t *testing.T) {
	config := &config.DynamicSLConfig{
		FirstMovePct:     0.01, // 1%
		TrailingStepPct:  0.005, // 0.5%
		StopMoveStepPct:  0.001, // 0.1%
	}

	entryPrice := 50000.0
	currentPrice := 50800.0 // 1.6% profit
	highestPrice := 50800.0
	currentSlPrice := 49000.0 // Still below breakeven
	firstMoveTriggered := true
	isLong := true

	shouldAdjust, newSlPrice, err := CalculateDynamicSL(
		entryPrice, currentPrice, highestPrice, currentSlPrice,
		firstMoveTriggered, isLong, config,
	)

	if err != nil {
		t.Fatalf("CalculateDynamicSL failed: %v", err)
	}
	if !shouldAdjust {
		t.Error("Expected shouldAdjust=true to move SL to breakeven")
	}

	expectedBreakeven := entryPrice * 1.001 // 50050
	if newSlPrice < expectedBreakeven-0.1 || newSlPrice > expectedBreakeven+0.1 {
		t.Errorf("Expected newSlPrice≈%.2f (breakeven), got %.2f", expectedBreakeven, newSlPrice)
	}
}

// TestCalculateDynamicSL_TrailingLogic 测试追踪止损逻辑 / Test trailing stop-loss logic
func TestCalculateDynamicSL_TrailingLogic(t *testing.T) {
	config := &config.DynamicSLConfig{
		FirstMovePct:     0.01, // 1%
		TrailingStepPct:  0.005, // 0.5%
		StopMoveStepPct:  0.001, // 0.1%
	}

	entryPrice := 50000.0
	highestPrice := 51250.0 // Previous highest
	currentPrice := 51506.25 // Price gained exactly 0.5% from highest
	currentSlPrice := 50050.0 // Already at breakeven
	firstMoveTriggered := true
	isLong := true

	shouldAdjust, newSlPrice, err := CalculateDynamicSL(
		entryPrice, currentPrice, highestPrice, currentSlPrice,
		firstMoveTriggered, isLong, config,
	)

	if err != nil {
		t.Fatalf("CalculateDynamicSL failed: %v", err)
	}
	if !shouldAdjust {
		t.Error("Expected shouldAdjust=true when price gains trailingStepPct")
	}

	// SL should move up by stopMoveStepPct (0.1%)
	expectedNewSl := currentSlPrice * 1.001 // 50100.05
	if newSlPrice < expectedNewSl-0.1 || newSlPrice > expectedNewSl+0.1 {
		t.Errorf("Expected newSlPrice≈%.2f, got %.2f", expectedNewSl, newSlPrice)
	}
}

// TestCalculateDynamicSL_ShortPosition 测试空头持仓逻辑 / Test short position logic
func TestCalculateDynamicSL_ShortPosition(t *testing.T) {
	config := &config.DynamicSLConfig{
		FirstMovePct:     0.01, // 1%
		TrailingStepPct:  0.005, // 0.5%
		StopMoveStepPct:  0.001, // 0.1%
	}

	entryPrice := 50000.0
	currentPrice := 49500.0 // 1% profit for short
	lowestPrice := 49500.0
	currentSlPrice := 51000.0 // Above entry
	firstMoveTriggered := false
	isLong := false

	shouldAdjust, newSlPrice, err := CalculateDynamicSL(
		entryPrice, currentPrice, lowestPrice, currentSlPrice,
		firstMoveTriggered, isLong, config,
	)

	if err != nil {
		t.Fatalf("CalculateDynamicSL failed: %v", err)
	}
	if !shouldAdjust {
		t.Error("Expected shouldAdjust=true when firstMove triggered for short")
	}

	expectedBreakeven := entryPrice * 0.999 // 49950
	if newSlPrice < expectedBreakeven-0.1 || newSlPrice > expectedBreakeven+0.1 {
		t.Errorf("Expected newSlPrice≈%.2f (breakeven for short), got %.2f", expectedBreakeven, newSlPrice)
	}
}

// TestCalculateDynamicSL_ShortTrailing 测试空头追踪止损 / Test short trailing stop
func TestCalculateDynamicSL_ShortTrailing(t *testing.T) {
	config := &config.DynamicSLConfig{
		FirstMovePct:     0.01, // 1%
		TrailingStepPct:  0.005, // 0.5%
		StopMoveStepPct:  0.001, // 0.1%
	}

	entryPrice := 50000.0
	currentPrice := 48500.0 // Price dropped 0.5% from lowest
	lowestPrice := 48750.0 // Previous lowest
	currentSlPrice := 49950.0 // Already at breakeven
	firstMoveTriggered := true
	isLong := false

	shouldAdjust, newSlPrice, err := CalculateDynamicSL(
		entryPrice, currentPrice, lowestPrice, currentSlPrice,
		firstMoveTriggered, isLong, config,
	)

	if err != nil {
		t.Fatalf("CalculateDynamicSL failed: %v", err)
	}
	if !shouldAdjust {
		t.Error("Expected shouldAdjust=true when price drops trailingStepPct for short")
	}

	// SL should move down by stopMoveStepPct (0.1%)
	expectedNewSl := currentSlPrice * 0.999 // 49900.05
	if newSlPrice < expectedNewSl-0.1 || newSlPrice > expectedNewSl+0.1 {
		t.Errorf("Expected newSlPrice≈%.2f, got %.2f", expectedNewSl, newSlPrice)
	}
}

// TestCalculateDynamicSL_ValidationErrors 测试输入验证 / Test input validation
func TestCalculateDynamicSL_ValidationErrors(t *testing.T) {
	defaultCfg := &config.DynamicSLConfig{
		FirstMovePct:     0.01,
		TrailingStepPct:  0.005,
		StopMoveStepPct:  0.001,
	}

	tests := []struct {
		name           string
		entryPrice     float64
		currentPrice   float64
		currentSlPrice float64
		cfg            *config.DynamicSLConfig
		expectError    bool
	}{
		{"Zero entry price", 0, 50000, 49000, defaultCfg, true},
		{"Negative entry price", -50000, 50000, 49000, defaultCfg, true},
		{"Zero current price", 50000, 0, 49000, defaultCfg, true},
		{"Zero SL price", 50000, 50000, 0, defaultCfg, true},
		{"Nil config", 50000, 50000, 49000, nil, true},
		{"Valid inputs", 50000, 50000, 49000, defaultCfg, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := CalculateDynamicSL(
				tt.entryPrice, tt.currentPrice, 50000, tt.currentSlPrice,
				false, true, tt.cfg,
			)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

// TestShouldAdjustSL_Wrapper 测试ShouldAdjustSL包装函数 / Test ShouldAdjustSL wrapper function
func TestShouldAdjustSL_Wrapper(t *testing.T) {
	config := &config.DynamicSLConfig{
		FirstMovePct:     0.01,
		TrailingStepPct:  0.005,
		StopMoveStepPct:  0.001,
	}

	tracker := &models.DynamicSLTracker{
		PositionKey:         "BTC-USDT-SWAP_long",
		InstId:              "BTC-USDT-SWAP",
		PosSide:             "long",
		EntryPrice:          50000.0,
		CurrentSlPrice:      49000.0,
		HighestPriceReached: 50000.0,
		LowestPriceReached:  0,
		FirstMoveTriggered:  false,
	}

	currentPrice := 50500.0 // 1% profit

	shouldAdjust, newSlPrice, err := ShouldAdjustSL(tracker, currentPrice, config)
	if err != nil {
		t.Fatalf("ShouldAdjustSL failed: %v", err)
	}

	if !shouldAdjust {
		t.Error("Expected shouldAdjust=true at 1% profit")
	}

	expectedBreakeven := 50050.0
	if newSlPrice < expectedBreakeven-0.1 || newSlPrice > expectedBreakeven+0.1 {
		t.Errorf("Expected newSlPrice≈%.2f, got %.2f", expectedBreakeven, newSlPrice)
	}
}

// TestShouldAdjustSL_InvalidTracker 测试无效追踪器验证 / Test invalid tracker validation
func TestShouldAdjustSL_InvalidTracker(t *testing.T) {
	config := &config.DynamicSLConfig{
		FirstMovePct:     0.01,
		TrailingStepPct:  0.005,
		StopMoveStepPct:  0.001,
	}

	// Empty position key
	tracker := &models.DynamicSLTracker{
		PositionKey:    "",
		InstId:         "BTC-USDT-SWAP",
		PosSide:        "long",
		EntryPrice:     50000.0,
		CurrentSlPrice: 49000.0,
	}

	_, _, err := ShouldAdjustSL(tracker, 50500.0, config)
	if err == nil {
		t.Error("Expected error for invalid tracker (empty PositionKey)")
	}
}
