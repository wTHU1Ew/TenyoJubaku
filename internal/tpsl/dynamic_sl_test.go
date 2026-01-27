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

// TestLoadOrCreateTracker_CreateNew 测试创建新的追踪器 / Test creating new tracker (V5.1)
func TestLoadOrCreateTracker_CreateNew(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	position := &models.Position{
		Instrument:   "BTC-USDT-SWAP",
		PositionSide: models.PositionSideLong,
		AveragePrice: 50000.0,
		Leverage:     10.0,
	}
	currentSlPrice := 49000.0
	leverage := 10.0

	tracker, err := LoadOrCreateTracker(ctx, storage, position, currentSlPrice, leverage)
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
	// V5.1: Verify new fields
	if tracker.InitialSlPrice != 49000.0 {
		t.Errorf("Expected InitialSlPrice=49000.0, got %.2f", tracker.InitialSlPrice)
	}
	if tracker.Leverage != 10.0 {
		t.Errorf("Expected Leverage=10.0, got %.2f", tracker.Leverage)
	}
	if tracker.HighestPriceReached != 50000.0 {
		t.Errorf("Expected HighestPriceReached=50000.0 (long position), got %.2f", tracker.HighestPriceReached)
	}
	if tracker.LowestPriceReached != 0 {
		t.Errorf("Expected LowestPriceReached=0 (long position), got %.2f", tracker.LowestPriceReached)
	}
	if tracker.BreakevenReached {
		t.Error("Expected BreakevenReached=false for new tracker")
	}
	if tracker.MoveCount != 0 {
		t.Errorf("Expected MoveCount=0 for new tracker, got %d", tracker.MoveCount)
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
		Leverage:     5.0,
	}
	currentSlPrice := 2950.0
	leverage := 5.0

	// Create tracker first time
	tracker1, err := LoadOrCreateTracker(ctx, storage, position, currentSlPrice, leverage)
	if err != nil {
		t.Fatalf("First LoadOrCreateTracker failed: %v", err)
	}
	originalID := tracker1.ID

	// Load tracker second time (should return existing)
	tracker2, err := LoadOrCreateTracker(ctx, storage, position, currentSlPrice, leverage)
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
		Leverage:     10.0,
	}
	currentSlPrice := 51000.0
	leverage := 10.0

	tracker, err := LoadOrCreateTracker(ctx, storage, position, currentSlPrice, leverage)
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

// TestUpdateTracker_LongHighestPrice 测试更新多头最高价格 / Test updating highest price for long (V5.1)
func TestUpdateTracker_LongHighestPrice(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	tracker := &models.DynamicSLTracker{
		PositionKey:         "BTC-USDT-SWAP_long",
		InstId:              "BTC-USDT-SWAP",
		PosSide:             "long",
		EntryPrice:          50000.0,
		CurrentSlPrice:      49000.0,
		InitialSlPrice:      49000.0,
		Leverage:            10.0,
		HighestPriceReached: 50000.0,
		LowestPriceReached:  0,
		BreakevenReached:    false,
		MoveCount:           0,
		CreatedAt:           time.Now().UTC(),
		LastUpdatedAt:       time.Now().UTC(),
	}

	// Insert tracker
	err := storage.InsertDynamicSLTracker(ctx, tracker)
	if err != nil {
		t.Fatalf("Failed to insert tracker: %v", err)
	}

	// Update with higher price (V5.1 - no config param)
	currentPrice := 50500.0
	updated, err := UpdateTracker(ctx, storage, tracker, currentPrice)
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

// TestUpdateTracker_ShortLowestPrice 测试更新空头最低价格 / Test updating lowest price for short (V5.1)
func TestUpdateTracker_ShortLowestPrice(t *testing.T) {
	storage := setupTestStorage(t)
	ctx := context.Background()

	tracker := &models.DynamicSLTracker{
		PositionKey:         "BTC-USDT-SWAP_short",
		InstId:              "BTC-USDT-SWAP",
		PosSide:             "short",
		EntryPrice:          50000.0,
		CurrentSlPrice:      51000.0,
		InitialSlPrice:      51000.0,
		Leverage:            10.0,
		HighestPriceReached: 0,
		LowestPriceReached:  50000.0,
		BreakevenReached:    false,
		MoveCount:           0,
		CreatedAt:           time.Now().UTC(),
		LastUpdatedAt:       time.Now().UTC(),
	}

	err := storage.InsertDynamicSLTracker(ctx, tracker)
	if err != nil {
		t.Fatalf("Failed to insert tracker: %v", err)
	}

	// Update with lower price (profit for short) - V5.1
	currentPrice := 49500.0
	updated, err := UpdateTracker(ctx, storage, tracker, currentPrice)
	if err != nil {
		t.Fatalf("UpdateTracker failed: %v", err)
	}

	if !updated {
		t.Error("Expected updated=true when price decreases (short)")
	}
	if tracker.LowestPriceReached != 49500.0 {
		t.Errorf("Expected LowestPriceReached=49500.0, got %.2f", tracker.LowestPriceReached)
	}
}

// TestCalculateDynamicSL_BeforeFirstMove 测试PhaseA - 盈利不足时不调整 / Test Phase A - no adjustment when profit insufficient (V5.1)
func TestCalculateDynamicSL_BeforeFirstMove(t *testing.T) {
	cfg := &config.DynamicSLConfig{
		ProfitStepPct:   0.01,  // 1% base × leverage
		SlMoveStepPct:   0.0101, // 1.01% base × leverage
		TrailingStepPct: 0.005,  // 0.5%
		StopMoveStepPct: 0.001,  // 0.1%
	}

	entryPrice := 50000.0
	currentPrice := 50040.0 // Only 0.08% price profit, with 10x = 0.8% account profit
	extremePrice := 50040.0
	currentSlPrice := 49000.0
	initialSlPrice := 49000.0
	leverage := 10.0
	breakevenReached := false
	isLong := true

	shouldAdjust, newSlPrice, phase, err := CalculateDynamicSL(
		entryPrice, currentPrice, extremePrice, currentSlPrice,
		initialSlPrice, leverage, breakevenReached, isLong, cfg,
	)

	if err != nil {
		t.Fatalf("CalculateDynamicSL failed: %v", err)
	}
	if shouldAdjust {
		t.Error("Expected shouldAdjust=false when account profit < threshold (10%)")
	}
	if newSlPrice != 0 {
		t.Errorf("Expected newSlPrice=0 when not adjusting, got %.2f", newSlPrice)
	}
	if phase != PhaseA {
		t.Errorf("Expected phase=PhaseA, got %s", phase)
	}
}

// TestCalculateDynamicSL_FirstMoveTriggered 测试PhaseA - 触发保本移动 / Test Phase A - trigger breakeven move (V5.1)
func TestCalculateDynamicSL_FirstMoveTriggered(t *testing.T) {
	cfg := &config.DynamicSLConfig{
		ProfitStepPct:   0.01,  // 1% base × 10 = 10% account profit threshold
		SlMoveStepPct:   0.0101, // 1.01% base × 10 = 10.1% SL move
		TrailingStepPct: 0.005,
		StopMoveStepPct: 0.001,
	}

	entryPrice := 50000.0
	currentPrice := 50500.0 // 1% price profit × 10 = 10% account profit
	extremePrice := 50500.0
	currentSlPrice := 49000.0
	initialSlPrice := 49000.0
	leverage := 10.0
	breakevenReached := false
	isLong := true

	shouldAdjust, newSlPrice, phase, err := CalculateDynamicSL(
		entryPrice, currentPrice, extremePrice, currentSlPrice,
		initialSlPrice, leverage, breakevenReached, isLong, cfg,
	)

	if err != nil {
		t.Fatalf("CalculateDynamicSL failed: %v", err)
	}
	if !shouldAdjust {
		t.Error("Expected shouldAdjust=true when account profit >= threshold")
	}
	if phase != PhaseA {
		t.Errorf("Expected phase=PhaseA, got %s", phase)
	}

	// SL should move up by 10.1% (slMoveStepPct × leverage)
	// 49000 × 1.101 = 53949 - but capped at breakeven (50050)
	expectedBreakeven := entryPrice * 1.001 // 50050
	if newSlPrice < expectedBreakeven-1 || newSlPrice > expectedBreakeven+1 {
		t.Errorf("Expected newSlPrice≈%.2f (capped at breakeven), got %.2f", expectedBreakeven, newSlPrice)
	}
}

// TestCalculateDynamicSL_TrailingLogic 测试PhaseB - 追踪止损逻辑 / Test Phase B - trailing logic (V5.1)
func TestCalculateDynamicSL_TrailingLogic(t *testing.T) {
	cfg := &config.DynamicSLConfig{
		ProfitStepPct:   0.01,
		SlMoveStepPct:   0.0101,
		TrailingStepPct: 0.005, // 0.5%
		StopMoveStepPct: 0.001, // 0.1%
	}

	entryPrice := 50000.0
	extremePrice := 51250.0               // Previous highest
	currentPrice := 51506.25              // Price gained exactly 0.5% from highest
	currentSlPrice := 50050.0             // Already at breakeven
	initialSlPrice := 49000.0
	leverage := 10.0
	breakevenReached := true // Already in Phase B
	isLong := true

	shouldAdjust, newSlPrice, phase, err := CalculateDynamicSL(
		entryPrice, currentPrice, extremePrice, currentSlPrice,
		initialSlPrice, leverage, breakevenReached, isLong, cfg,
	)

	if err != nil {
		t.Fatalf("CalculateDynamicSL failed: %v", err)
	}
	if !shouldAdjust {
		t.Error("Expected shouldAdjust=true when price gains trailingStepPct")
	}
	if phase != PhaseB {
		t.Errorf("Expected phase=PhaseB, got %s", phase)
	}

	// SL should move up by stopMoveStepPct (0.1%)
	expectedNewSl := currentSlPrice * 1.001 // 50100.05
	if newSlPrice < expectedNewSl-0.1 || newSlPrice > expectedNewSl+0.1 {
		t.Errorf("Expected newSlPrice≈%.2f, got %.2f", expectedNewSl, newSlPrice)
	}
}

// TestCalculateDynamicSL_ShortPosition 测试空头持仓PhaseA / Test short position Phase A (V5.1)
func TestCalculateDynamicSL_ShortPosition(t *testing.T) {
	cfg := &config.DynamicSLConfig{
		ProfitStepPct:   0.01, // 1% base × 10 = 10%
		SlMoveStepPct:   0.0101,
		TrailingStepPct: 0.005,
		StopMoveStepPct: 0.001,
	}

	entryPrice := 50000.0
	currentPrice := 49500.0 // 1% price profit for short × 10 = 10% account profit
	lowestPrice := 49500.0
	currentSlPrice := 51000.0 // Above entry
	initialSlPrice := 51000.0
	leverage := 10.0
	breakevenReached := false
	isLong := false

	shouldAdjust, newSlPrice, phase, err := CalculateDynamicSL(
		entryPrice, currentPrice, lowestPrice, currentSlPrice,
		initialSlPrice, leverage, breakevenReached, isLong, cfg,
	)

	if err != nil {
		t.Fatalf("CalculateDynamicSL failed: %v", err)
	}
	if !shouldAdjust {
		t.Error("Expected shouldAdjust=true when Phase A triggered for short")
	}
	if phase != PhaseA {
		t.Errorf("Expected phase=PhaseA, got %s", phase)
	}

	// For short, breakeven is entry × 0.999 = 49950
	expectedBreakeven := entryPrice * 0.999
	if newSlPrice < expectedBreakeven-1 || newSlPrice > expectedBreakeven+1 {
		t.Errorf("Expected newSlPrice≈%.2f (capped at breakeven for short), got %.2f", expectedBreakeven, newSlPrice)
	}
}

// TestCalculateDynamicSL_ShortTrailing 测试空头追踪止损PhaseB / Test short trailing Phase B (V5.1)
func TestCalculateDynamicSL_ShortTrailing(t *testing.T) {
	cfg := &config.DynamicSLConfig{
		ProfitStepPct:   0.01,
		SlMoveStepPct:   0.0101,
		TrailingStepPct: 0.005, // 0.5%
		StopMoveStepPct: 0.001, // 0.1%
	}

	entryPrice := 50000.0
	lowestPrice := 48750.0  // Previous lowest
	currentPrice := 48506.0 // Price dropped ~0.5% from lowest
	currentSlPrice := 49950.0 // Already at breakeven
	initialSlPrice := 51000.0
	leverage := 10.0
	breakevenReached := true
	isLong := false

	shouldAdjust, newSlPrice, phase, err := CalculateDynamicSL(
		entryPrice, currentPrice, lowestPrice, currentSlPrice,
		initialSlPrice, leverage, breakevenReached, isLong, cfg,
	)

	if err != nil {
		t.Fatalf("CalculateDynamicSL failed: %v", err)
	}
	if !shouldAdjust {
		t.Error("Expected shouldAdjust=true when price drops trailingStepPct for short")
	}
	if phase != PhaseB {
		t.Errorf("Expected phase=PhaseB, got %s", phase)
	}

	// SL should move down by stopMoveStepPct (0.1%)
	expectedNewSl := currentSlPrice * 0.999 // 49900.05
	if newSlPrice < expectedNewSl-0.1 || newSlPrice > expectedNewSl+0.1 {
		t.Errorf("Expected newSlPrice≈%.2f, got %.2f", expectedNewSl, newSlPrice)
	}
}

// TestCalculateDynamicSL_ValidationErrors 测试输入验证 / Test input validation (V5.1)
func TestCalculateDynamicSL_ValidationErrors(t *testing.T) {
	defaultCfg := &config.DynamicSLConfig{
		ProfitStepPct:   0.01,
		SlMoveStepPct:   0.0101,
		TrailingStepPct: 0.005,
		StopMoveStepPct: 0.001,
	}

	tests := []struct {
		name           string
		entryPrice     float64
		currentPrice   float64
		currentSlPrice float64
		initialSlPrice float64
		leverage       float64
		cfg            *config.DynamicSLConfig
		expectError    bool
	}{
		{"Zero entry price", 0, 50000, 49000, 49000, 10, defaultCfg, true},
		{"Negative entry price", -50000, 50000, 49000, 49000, 10, defaultCfg, true},
		{"Zero current price", 50000, 0, 49000, 49000, 10, defaultCfg, true},
		{"Zero SL price", 50000, 50000, 0, 49000, 10, defaultCfg, true},
		{"Zero initial SL price", 50000, 50000, 49000, 0, 10, defaultCfg, true},
		{"Leverage < 1", 50000, 50000, 49000, 49000, 0.5, defaultCfg, true},
		{"Nil config", 50000, 50000, 49000, 49000, 10, nil, true},
		{"Valid inputs", 50000, 50000, 49000, 49000, 10, defaultCfg, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, _, err := CalculateDynamicSL(
				tt.entryPrice, tt.currentPrice, 50000, tt.currentSlPrice,
				tt.initialSlPrice, tt.leverage, false, true, tt.cfg,
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

// TestShouldAdjustSL_Wrapper 测试ShouldAdjustSL包装函数 / Test ShouldAdjustSL wrapper function (V5.1)
func TestShouldAdjustSL_Wrapper(t *testing.T) {
	cfg := &config.DynamicSLConfig{
		ProfitStepPct:   0.01,
		SlMoveStepPct:   0.0101,
		TrailingStepPct: 0.005,
		StopMoveStepPct: 0.001,
	}

	tracker := &models.DynamicSLTracker{
		PositionKey:         "BTC-USDT-SWAP_long",
		InstId:              "BTC-USDT-SWAP",
		PosSide:             "long",
		EntryPrice:          50000.0,
		CurrentSlPrice:      49000.0,
		InitialSlPrice:      49000.0,
		Leverage:            10.0,
		HighestPriceReached: 50000.0,
		LowestPriceReached:  0,
		BreakevenReached:    false,
		MoveCount:           0,
	}

	currentPrice := 50500.0 // 1% price profit × 10 = 10% account profit

	shouldAdjust, newSlPrice, phase, err := ShouldAdjustSL(tracker, currentPrice, cfg)
	if err != nil {
		t.Fatalf("ShouldAdjustSL failed: %v", err)
	}

	if !shouldAdjust {
		t.Error("Expected shouldAdjust=true at 10% account profit")
	}
	if phase != PhaseA {
		t.Errorf("Expected phase=PhaseA, got %s", phase)
	}

	// Should be capped at breakeven
	expectedBreakeven := 50050.0
	if newSlPrice < expectedBreakeven-1 || newSlPrice > expectedBreakeven+1 {
		t.Errorf("Expected newSlPrice≈%.2f, got %.2f", expectedBreakeven, newSlPrice)
	}
}

// TestShouldAdjustSL_InvalidTracker 测试无效追踪器验证 / Test invalid tracker validation (V5.1)
func TestShouldAdjustSL_InvalidTracker(t *testing.T) {
	cfg := &config.DynamicSLConfig{
		ProfitStepPct:   0.01,
		SlMoveStepPct:   0.0101,
		TrailingStepPct: 0.005,
		StopMoveStepPct: 0.001,
	}

	// Empty position key
	tracker := &models.DynamicSLTracker{
		PositionKey:    "",
		InstId:         "BTC-USDT-SWAP",
		PosSide:        "long",
		EntryPrice:     50000.0,
		CurrentSlPrice: 49000.0,
		InitialSlPrice: 49000.0,
		Leverage:       10.0,
	}

	_, _, _, err := ShouldAdjustSL(tracker, 50500.0, cfg)
	if err == nil {
		t.Error("Expected error for invalid tracker (empty PositionKey)")
	}
}

// TestIsBreakevenReached 测试保本位检测 / Test breakeven detection (V5.1)
func TestIsBreakevenReached(t *testing.T) {
	tests := []struct {
		name           string
		posSide        string
		entryPrice     float64
		currentSlPrice float64
		expected       bool
	}{
		{"Long - below breakeven", "long", 50000, 49000, false},
		{"Long - at breakeven", "long", 50000, 50050, true},
		{"Long - above breakeven", "long", 50000, 51000, true},
		{"Short - above breakeven", "short", 50000, 51000, false},
		{"Short - at breakeven", "short", 50000, 49950, true},
		{"Short - below breakeven", "short", 50000, 49000, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := &models.DynamicSLTracker{
				PosSide:        tt.posSide,
				EntryPrice:     tt.entryPrice,
				CurrentSlPrice: tt.currentSlPrice,
			}

			result := IsBreakevenReached(tracker)
			if result != tt.expected {
				t.Errorf("Expected IsBreakevenReached=%v, got %v", tt.expected, result)
			}
		})
	}
}

// TestGetDistanceToBreakeven 测试保本位距离计算 / Test distance to breakeven calculation (V5.1)
func TestGetDistanceToBreakeven(t *testing.T) {
	tests := []struct {
		name           string
		posSide        string
		entryPrice     float64
		currentSlPrice float64
		expectedSign   int // 1 = positive (need to move), -1 = negative (past breakeven), 0 = at breakeven
	}{
		{"Long - below breakeven", "long", 50000, 49000, 1},
		{"Long - at breakeven", "long", 50000, 50050, 0},
		{"Long - above breakeven", "long", 50000, 51000, -1},
		{"Short - above breakeven", "short", 50000, 51000, 1},
		{"Short - at breakeven", "short", 50000, 49950, 0},
		{"Short - below breakeven", "short", 50000, 49000, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tracker := &models.DynamicSLTracker{
				PosSide:        tt.posSide,
				EntryPrice:     tt.entryPrice,
				CurrentSlPrice: tt.currentSlPrice,
			}

			distance := GetDistanceToBreakeven(tracker)

			if tt.expectedSign == 1 && distance <= 0 {
				t.Errorf("Expected positive distance (need to move), got %.6f", distance)
			}
			if tt.expectedSign == -1 && distance >= 0 {
				t.Errorf("Expected negative distance (past breakeven), got %.6f", distance)
			}
			if tt.expectedSign == 0 && (distance > 0.0001 || distance < -0.0001) {
				t.Errorf("Expected distance ≈ 0, got %.6f", distance)
			}
		})
	}
}

// TestPhaseA_MultipleMoves 测试PhaseA多次移动 / Test Phase A multiple moves before breakeven (V5.1)
func TestPhaseA_MultipleMoves(t *testing.T) {
	cfg := &config.DynamicSLConfig{
		ProfitStepPct:   0.01,  // 1% base
		SlMoveStepPct:   0.0101, // 1.01% base
		TrailingStepPct: 0.005,
		StopMoveStepPct: 0.001,
	}

	// With 2x leverage, 1% price profit = 2% account profit
	// Threshold = 1% × 2 = 2%
	// SL move = 1.01% × 2 = 2.02%
	entryPrice := 50000.0
	currentSlPrice := 49000.0 // Initial SL at -2%
	initialSlPrice := 49000.0
	leverage := 2.0
	breakevenReached := false

	// First trigger: price up 1% → 2% account profit
	currentPrice := 50500.0
	shouldAdjust, newSlPrice, phase, err := CalculateDynamicSL(
		entryPrice, currentPrice, currentPrice, currentSlPrice,
		initialSlPrice, leverage, breakevenReached, true, cfg,
	)

	if err != nil {
		t.Fatalf("CalculateDynamicSL failed: %v", err)
	}
	if !shouldAdjust {
		t.Error("Expected shouldAdjust=true for first Phase A move")
	}
	if phase != PhaseA {
		t.Errorf("Expected phase=PhaseA, got %s", phase)
	}

	// SL should move up by 2.02% from 49000 → ~49989
	// But should be capped at breakeven (50050)
	expectedBreakeven := entryPrice * 1.001
	if newSlPrice > expectedBreakeven+1 {
		t.Errorf("Expected newSlPrice <= %.2f (breakeven cap), got %.2f", expectedBreakeven, newSlPrice)
	}
	// Should be higher than current SL
	if newSlPrice <= currentSlPrice {
		t.Errorf("Expected newSlPrice > %.2f, got %.2f", currentSlPrice, newSlPrice)
	}
}
