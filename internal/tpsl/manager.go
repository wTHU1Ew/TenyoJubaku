package tpsl

import (
	"fmt"
	"strconv"

	"github.com/wTHU1Ew/TenyoJubaku/internal/config"
	"github.com/wTHU1Ew/TenyoJubaku/internal/logger"
	"github.com/wTHU1Ew/TenyoJubaku/internal/okx"
	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

// Manager TPSL管理器 / TPSL manager
// 负责分析持仓TPSL覆盖情况并下单TPSL订单
// Responsible for analyzing position TPSL coverage and placing TPSL orders
type Manager struct {
	config    *config.TPSLConfig
	okxClient *okx.Client
	logger    *logger.Logger
}

// TPSLPrices TPSL价格 / TPSL prices
type TPSLPrices struct {
	TpPrice float64
	SlPrice float64
}

// CoverageSummary 覆盖情况汇总 / Coverage summary
type CoverageSummary struct {
	TotalChecked      int
	FullyCovered      int
	PartiallyCovered  int
	NotCovered        int
	OrdersPlaced      int
	PlacementFailures int
}

// New 创建TPSL管理器 / Create TPSL manager
// 初始化TPSL管理器实例
// Initialize TPSL manager instance
//
// Parameters:
//   - config: TPSL configuration
//   - okxClient: OKX API client
//   - logger: Logger instance
//
// Returns:
//   - *Manager: TPSL管理器实例 / TPSL manager instance
func New(config *config.TPSLConfig, okxClient *okx.Client, logger *logger.Logger) *Manager {
	return &Manager{
		config:    config,
		okxClient: okxClient,
		logger:    logger,
	}
}

// AnalyzeAndPlaceTPSL 分析持仓并下单TPSL / Analyze positions and place TPSL orders
// 主要入口点：分析所有持仓的TPSL覆盖情况，并为未覆盖的持仓下单TPSL订单
// Main entry point: analyze all positions' TPSL coverage and place TPSL orders for uncovered positions
//
// Parameters:
//   - positions: 持仓列表 / List of positions
//
// Returns:
//   - *CoverageSummary: 覆盖情况汇总 / Coverage summary
//   - error: 处理失败时返回错误 / Error on processing failure
func (m *Manager) AnalyzeAndPlaceTPSL(positions []*models.Position) (*CoverageSummary, error) {
	summary := &CoverageSummary{}

	// Handle empty positions list
	if len(positions) == 0 {
		m.logger.Info("No open positions, skipping TPSL check")
		return summary, nil
	}

	m.logger.Info("Starting TPSL analysis for %d positions", len(positions))

	// Query pending algo orders
	algoOrders, err := m.okxClient.GetPendingAlgoOrders("conditional")
	if err != nil {
		return nil, fmt.Errorf("failed to get pending algo orders: %w", err)
	}

	m.logger.Debug("Retrieved %d pending conditional algo orders", len(algoOrders.Data))

	// Analyze each position
	for _, position := range positions {
		summary.TotalChecked++

		// Analyze coverage
		uncoveredSize := m.analyzeCoverage(position, algoOrders.Data)

		if uncoveredSize <= 0.000001 { // Effectively zero (account for float precision)
			m.logger.Debug("Position %s (%s) fully covered by TPSL", position.Instrument, position.PositionSide)
			summary.FullyCovered++
			continue
		}

		// Check if partial coverage
		if uncoveredSize < position.PositionSize {
			m.logger.Info("Position %s (%s) partially covered, uncovered size: %.8f",
				position.Instrument, position.PositionSide, uncoveredSize)
			summary.PartiallyCovered++
		} else {
			m.logger.Info("Position %s (%s) has no TPSL coverage, size: %.8f",
				position.Instrument, position.PositionSide, position.PositionSize)
			summary.NotCovered++
		}

		// Calculate TPSL prices
		prices, err := m.calculateTPSLPrices(position)
		if err != nil {
			m.logger.Error("Failed to calculate TPSL prices for %s: %v", position.Instrument, err)
			summary.PlacementFailures++
			continue
		}

		// Place TPSL order with current price validation
		err = m.placeTPSLOrderWithValidation(position, uncoveredSize, prices)
		if err != nil {
			m.logger.Error("Failed to place TPSL for %s: %v", position.Instrument, err)
			summary.PlacementFailures++
			continue
		}

		summary.OrdersPlaced++
	}

	m.logger.Info("TPSL check complete: checked=%d, fully_covered=%d, partially_covered=%d, not_covered=%d, orders_placed=%d, failures=%d",
		summary.TotalChecked, summary.FullyCovered, summary.PartiallyCovered,
		summary.NotCovered, summary.OrdersPlaced, summary.PlacementFailures)

	return summary, nil
}

// analyzeCoverage 分析持仓TPSL覆盖情况 / Analyze position TPSL coverage
// 计算持仓的未覆盖大小
// Calculate uncovered size of position
//
// 修改说明 / Modification Note:
// 由于TP和SL现在是两个独立的订单，我们需要检查是否同时存在TP和SL订单。
// 只有同时有TP和SL的部分才算完全覆盖。
// Since TP and SL are now two separate orders, we need to check if both TP and SL exist.
// Only the portion covered by BOTH TP and SL orders is considered fully covered.
//
// Parameters:
//   - position: 持仓信息 / Position information
//   - algoOrders: 算法订单列表 / List of algo orders
//
// Returns:
//   - float64: 未覆盖的持仓大小 / Uncovered position size
func (m *Manager) analyzeCoverage(position *models.Position, algoOrders []okx.AlgoOrder) float64 {
	maxTpSize := 0.0
	maxSlSize := 0.0
	tpCount := 0
	slCount := 0

	// Filter matching algo orders and track TP and SL separately
	// We need BOTH TP and SL to consider a position covered
	for _, order := range algoOrders {
		if m.matchesPosition(&order, position) {
			// Parse order size
			size, err := strconv.ParseFloat(order.Sz, 64)
			if err != nil {
				m.logger.Warn("Failed to parse algo order size '%s' for order %s: %v", order.Sz, order.AlgoId, err)
				continue
			}

			// Determine if this is TP or SL order
			hasTp := order.TpTriggerPx != "" && order.TpTriggerPx != "0"
			hasSl := order.SlTriggerPx != "" && order.SlTriggerPx != "0"

			if hasTp {
				tpCount++
				if size > maxTpSize {
					maxTpSize = size
				}
				m.logger.Debug("Found Take-Profit order %s with size %.8f for position %s",
					order.AlgoId, size, position.Instrument)
			}
			if hasSl {
				slCount++
				if size > maxSlSize {
					maxSlSize = size
				}
				m.logger.Debug("Found Stop-Loss order %s with size %.8f for position %s",
					order.AlgoId, size, position.Instrument)
			}
		}
	}

	// Only the portion covered by BOTH TP and SL is considered covered
	// If either TP or SL is missing, the position is not properly covered
	coveredSize := 0.0
	if tpCount > 0 && slCount > 0 {
		// Use the minimum of TP and SL sizes (conservative approach)
		// because only the portion covered by BOTH is truly protected
		if maxTpSize < maxSlSize {
			coveredSize = maxTpSize
		} else {
			coveredSize = maxSlSize
		}
	} else if tpCount > 0 {
		m.logger.Warn("Position %s has TP orders but NO SL orders - not considered covered!", position.Instrument)
	} else if slCount > 0 {
		m.logger.Warn("Position %s has SL orders but NO TP orders - not considered covered!", position.Instrument)
	}

	uncoveredSize := position.PositionSize - coveredSize
	if uncoveredSize < 0 {
		uncoveredSize = 0 // Shouldn't happen, but handle gracefully
	}

	m.logger.Info("Position %s coverage: total=%.8f, TP_covered=%.8f (count:%d), SL_covered=%.8f (count:%d), final_covered=%.8f, uncovered=%.8f",
		position.Instrument, position.PositionSize, maxTpSize, tpCount, maxSlSize, slCount, coveredSize, uncoveredSize)

	return uncoveredSize
}

// matchesPosition 判断算法订单是否匹配持仓 / Check if algo order matches position
// 检查算法订单的交易对和持仓方向是否与持仓匹配
// Check if algo order's instrument and position side match the position
//
// Parameters:
//   - order: 算法订单 / Algo order
//   - position: 持仓信息 / Position information
//
// Returns:
//   - bool: 是否匹配 / Whether it matches
func (m *Manager) matchesPosition(order *okx.AlgoOrder, position *models.Position) bool {
	// Check instrument
	if order.InstId != position.Instrument {
		return false
	}

	// Check position side
	if order.PosSide != position.PositionSide.String() {
		return false
	}

	// Check order type (must be conditional TPSL)
	if order.OrdType != "conditional" {
		return false
	}

	// Check state (must be live)
	if order.State != "live" {
		return false
	}

	return true
}

// calculateTPSLPrices 计算TPSL价格 / Calculate TPSL prices
// 根据持仓入场价、波动率百分比和盈亏比计算止盈止损价格
// Calculate stop-loss and take-profit prices based on entry price, volatility percentage, and profit-loss ratio
//
// 计算逻辑 / Calculation Logic:
// - 止损距离 = 入场价 × 波动率百分比 (不考虑杠杆)
// - 止盈距离 = 入场价 × 波动率百分比 × 盈亏比
// 例如: 入场价$100, 波动率1%, 盈亏比5:1
//   多头: SL=$99 (-1%), TP=$105 (+5%)
//   空头: SL=$101 (+1%), TP=$95 (-5%)
//
// Parameters:
//   - position: 持仓信息 / Position information
//
// Returns:
//   - *TPSLPrices: TPSL价格 / TPSL prices
//   - error: 计算失败时返回错误 / Error on calculation failure
func (m *Manager) calculateTPSLPrices(position *models.Position) (*TPSLPrices, error) {
	entryPrice := position.AveragePrice
	volatilityPct := m.config.VolatilityPct
	plRatio := m.config.ProfitLossRatio

	if entryPrice <= 0 {
		return nil, fmt.Errorf("invalid entry price: %.8f", entryPrice)
	}

	// Calculate SL distance (percentage of entry price, NOT considering leverage)
	slDistance := entryPrice * volatilityPct

	// Calculate TP distance (SL distance multiplied by profit-loss ratio)
	tpDistance := entryPrice * volatilityPct * plRatio

	var tpPrice, slPrice float64

	// Determine if position is long or short
	isLong := m.isLongPosition(position)

	if isLong {
		// Long position: SL below entry, TP above entry
		slPrice = entryPrice - slDistance
		tpPrice = entryPrice + tpDistance
	} else {
		// Short position: SL above entry, TP below entry
		slPrice = entryPrice + slDistance
		tpPrice = entryPrice - tpDistance
	}

	// Validate prices
	if slPrice <= 0 || tpPrice <= 0 {
		return nil, fmt.Errorf("invalid calculated prices: SL=%.8f, TP=%.8f", slPrice, tpPrice)
	}

	m.logger.Debug("Calculated TPSL for %s (%s): entry=%.8f, volatility=%.2f%%, SL=%.8f, TP=%.8f",
		position.Instrument, position.PositionSide, entryPrice, volatilityPct*100, slPrice, tpPrice)

	return &TPSLPrices{
		TpPrice: tpPrice,
		SlPrice: slPrice,
	}, nil
}

// isLongPosition 判断是否为多头持仓 / Check if position is long
// 根据持仓方向判断是否为多头
// Determine if position is long based on position side
//
// Parameters:
//   - position: 持仓信息 / Position information
//
// Returns:
//   - bool: 是否为多头 / Whether it's a long position
func (m *Manager) isLongPosition(position *models.Position) bool {
	// In hedge mode, position side is explicitly "long" or "short"
	if position.PositionSide == models.PositionSideLong {
		return true
	}
	if position.PositionSide == models.PositionSideShort {
		return false
	}

	// In one-way (net) mode, determine from position size
	// Positive size = long, negative size = short
	return position.PositionSize > 0
}

// placeTPSLOrder 下单TPSL订单 / Place TPSL order
// 为持仓下单止盈止损订单（分成两个独立订单）
// Place take-profit and stop-loss orders for position (as two separate orders)
//
// 修改说明 / Modification Note:
// 1. 由于OKX API在某些情况下同时发送TP和SL时可能只执行SL，
//    因此将止盈和止损分成两个独立的订单分别下单。
// 2. 检查当前价格是否已超过预期止盈位置，如果已超过则跳过止盈订单设置。
// 1. Due to OKX API limitation where only SL is executed when both TP and SL are sent together,
//    we now place take-profit and stop-loss as two separate orders.
// 2. Check if current price has already passed the expected TP price, skip TP order if so.
//
// Parameters:
//   - position: 持仓信息 / Position information
//   - size: 订单大小 / Order size
//   - prices: TPSL价格 / TPSL prices
//
// Returns:
//   - error: 下单失败时返回错误 / Error on placement failure
//
// NOTE: This function signature is kept for backward compatibility.
// The actual implementation now includes current price validation.
// See placeTPSLOrderWithValidation() for the full implementation.

// formatFloat 格式化浮点数为字符串 / Format float to string
// 将浮点数格式化为字符串，保留足够精度
// Format float to string with sufficient precision
//
// Parameters:
//   - f: 浮点数 / Float number
//
// Returns:
//   - string: 格式化后的字符串 / Formatted string
func formatFloat(f float64) string {
	// Remove trailing zeros
	s := strconv.FormatFloat(f, 'f', 8, 64)
	// Trim trailing zeros after decimal point
	if len(s) > 0 && (s[len(s)-1] == '0' || s[len(s)-1] == '.') {
		s = trimTrailingZeros(s)
	}
	return s
}

// trimTrailingZeros 去除尾随零 / Trim trailing zeros
func trimTrailingZeros(s string) string {
	// Find decimal point
	dotIndex := -1
	for i, ch := range s {
		if ch == '.' {
			dotIndex = i
			break
		}
	}

	if dotIndex == -1 {
		return s // No decimal point
	}

	// Trim zeros from the end
	end := len(s) - 1
	for end > dotIndex && s[end] == '0' {
		end--
	}

	// If only decimal point left, remove it too
	if end == dotIndex {
		return s[:dotIndex]
	}

	return s[:end+1]
}

// getCurrentMarketPrice 获取当前市场价格 / Get current market price from OKX ticker API
// 从OKX ticker API获取指定交易对的当前价格
// Fetch current price for the specified instrument from OKX ticker API
//
// Parameters:
//   - instId: 交易对ID / Instrument ID (e.g., "BTC-USDT-SWAP")
//
// Returns:
//   - float64: 当前市场价格 / Current market price
//   - error: 获取失败时返回错误 / Error on failure
func (m *Manager) getCurrentMarketPrice(instId string) (float64, error) {
	// Query OKX ticker API
	resp, err := m.okxClient.GetTicker(instId)
	if err != nil {
		return 0, fmt.Errorf("failed to get ticker for %s: %w", instId, err)
	}

	// Parse last price
	if len(resp.Data) == 0 {
		return 0, fmt.Errorf("no ticker data returned for %s", instId)
	}

	lastPrice, err := strconv.ParseFloat(resp.Data[0].Last, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse last price '%s': %w", resp.Data[0].Last, err)
	}

	return lastPrice, nil
}

// adjustTPSLPricesWithCurrentPrice 根据当前价格调整止盈止损价格 / Adjust TP/SL prices based on current market price
// 检查当前价格是否已经超过预期的止盈/止损位置，如果是则使用当前价格
// Check if current price has exceeded expected TP/SL levels, use current price if so
//
// Parameters:
//   - position: 持仓信息 / Position information
//   - prices: 原始计算的TPSL价格 / Originally calculated TPSL prices
//   - currentPrice: 当前市场价格 / Current market price
//
// Returns:
//   - *TPSLPrices: 调整后的TPSL价格 / Adjusted TPSL prices
//   - bool: 是否跳过止盈订单 / Whether to skip TP order (if price moved too far)
//   - bool: 是否跳过止损订单 / Whether to skip SL order (if price moved too far)
func (m *Manager) adjustTPSLPricesWithCurrentPrice(position *models.Position, prices *TPSLPrices, currentPrice float64) (*TPSLPrices, bool, bool) {
	isLong := m.isLongPosition(position)
	adjustedPrices := &TPSLPrices{
		TpPrice: prices.TpPrice,
		SlPrice: prices.SlPrice,
	}
	skipTP := false
	skipSL := false

	m.logger.Debug("Checking TP/SL prices for %s (%s): entry=%.8f, current=%.8f, expected_TP=%.8f, expected_SL=%.8f",
		position.Instrument, position.PositionSide, position.AveragePrice, currentPrice, prices.TpPrice, prices.SlPrice)

	if isLong {
		// Long position: TP above entry, SL below entry

		// Check TP: if current price >= expected TP price
		if currentPrice >= prices.TpPrice {
			m.logger.Warn("Position %s (long): Current price %.8f has reached or exceeded expected TP %.8f",
				position.Instrument, currentPrice, prices.TpPrice)
			// For long position: TP must be ABOVE current price
			// Set TP slightly above current price (0.1% higher to ensure it's above)
			adjustedPrice := currentPrice * 1.001
			m.logger.Info("Adjusting TP price to slightly above current price: %.8f → %.8f (current: %.8f)",
				prices.TpPrice, adjustedPrice, currentPrice)
			adjustedPrices.TpPrice = adjustedPrice
		}

		// Check SL: if current price <= expected SL price
		if currentPrice <= prices.SlPrice {
			m.logger.Warn("Position %s (long): Current price %.8f has hit or passed expected SL %.8f!",
				position.Instrument, currentPrice, prices.SlPrice)
			// For long position: SL must be BELOW current price
			// Set SL slightly below current price (0.1% lower) to ensure it triggers
			// User accepts slightly more loss to ensure SL is set
			adjustedPrice := currentPrice * 0.999
			m.logger.Info("Adjusting SL price to slightly below current price: %.8f → %.8f (current: %.8f)",
				prices.SlPrice, adjustedPrice, currentPrice)
			m.logger.Warn("ALERT: Setting emergency SL at current price - position already in loss beyond expected SL")
			adjustedPrices.SlPrice = adjustedPrice
		}

	} else {
		// Short position: TP below entry, SL above entry

		// Check TP: if current price <= expected TP price
		if currentPrice <= prices.TpPrice {
			m.logger.Warn("Position %s (short): Current price %.8f has reached or exceeded expected TP %.8f",
				position.Instrument, currentPrice, prices.TpPrice)
			// For short position: TP must be BELOW current price
			// Set TP slightly below current price (0.1% lower to ensure it's below)
			adjustedPrice := currentPrice * 0.999
			m.logger.Info("Adjusting TP price to slightly below current price: %.8f → %.8f (current: %.8f)",
				prices.TpPrice, adjustedPrice, currentPrice)
			adjustedPrices.TpPrice = adjustedPrice
		}

		// Check SL: if current price >= expected SL price
		if currentPrice >= prices.SlPrice {
			m.logger.Warn("Position %s (short): Current price %.8f has hit or passed expected SL %.8f!",
				position.Instrument, currentPrice, prices.SlPrice)
			// For short position: SL must be ABOVE current price
			// Set SL slightly above current price (0.1% higher) to ensure it triggers
			// User accepts slightly more loss to ensure SL is set
			adjustedPrice := currentPrice * 1.001
			m.logger.Info("Adjusting SL price to slightly above current price: %.8f → %.8f (current: %.8f)",
				prices.SlPrice, adjustedPrice, currentPrice)
			m.logger.Warn("ALERT: Setting emergency SL at current price - position already in loss beyond expected SL")
			adjustedPrices.SlPrice = adjustedPrice
		}
	}

	return adjustedPrices, skipTP, skipSL
}

// placeTPSLOrderWithValidation 下单TPSL订单（带价格验证）/ Place TPSL order with price validation
// 为持仓下单止盈止损订单（分成两个独立订单），并根据当前价格调整
// Place take-profit and stop-loss orders for position (as two separate orders) with current price adjustment
//
// 功能说明 / Features:
// 1. 从OKX API获取当前市场价格
// 2. 对比当前价格和预期止盈/止损价格
// 3. 如果当前价格已超过预期止盈价格，使用当前价格作为止盈价
// 4. 将止盈和止损分成两个独立订单下单
//
// 1. Fetch current market price from OKX API
// 2. Compare current price with expected TP/SL prices
// 3. If current price has exceeded expected TP, use current price as TP
// 4. Place TP and SL as two separate orders
//
// Parameters:
//   - position: 持仓信息 / Position information
//   - size: 订单大小 / Order size
//   - prices: TPSL价格 / TPSL prices
//
// Returns:
//   - error: 下单失败时返回错误 / Error on placement failure
func (m *Manager) placeTPSLOrderWithValidation(position *models.Position, size float64, prices *TPSLPrices) error {
	// Get current market price
	currentPrice, err := m.getCurrentMarketPrice(position.Instrument)
	if err != nil {
		m.logger.Warn("Failed to get current market price for %s: %v, proceeding with calculated prices", position.Instrument, err)
		// Fallback to original placeTPSLOrder without validation
		return m.placeTPSLOrderOriginal(position, size, prices)
	}

	// Adjust TP/SL prices based on current price
	adjustedPrices, skipTP, skipSL := m.adjustTPSLPricesWithCurrentPrice(position, prices, currentPrice)

	// Determine order side (opposite of position)
	isLong := m.isLongPosition(position)
	var orderSide string
	if isLong {
		orderSide = "sell" // Close long position
	} else {
		orderSide = "buy" // Close short position
	}

	m.logger.Info("Placing TPSL orders for %s (%s): TP=%.8f (adjusted: %v), SL=%.8f, current=%.8f",
		position.Instrument, position.PositionSide, adjustedPrices.TpPrice,
		adjustedPrices.TpPrice != prices.TpPrice, adjustedPrices.SlPrice, currentPrice)

	var tpAlgoId string

	// Determine trade mode from position
	tdMode := position.MarginMode.String()
	if tdMode == "" {
		tdMode = models.MarginModeCross.String() // Default to cross if not specified
	}

	// Place Take-Profit order (if not skipped)
	if !skipTP {
		tpReq := okx.AlgoOrderRequest{
			InstId:          position.Instrument,
			TdMode:          tdMode,
			Side:            orderSide,
			PosSide:         position.PositionSide.String(),
			OrdType:         "conditional",
			Sz:              formatFloat(size),
			TpTriggerPx:     formatFloat(adjustedPrices.TpPrice),
			TpOrdPx:         "-1", // Market order
			TpTriggerPxType: "last",
			ReduceOnly:      true,
		}

		m.logger.Debug("Placing Take-Profit order for %s (%s): TP=%.8f", position.Instrument, position.PositionSide, adjustedPrices.TpPrice)

		tpResp, err := m.okxClient.PlaceAlgoOrder(tpReq)
		if err != nil {
			return fmt.Errorf("Take-Profit order failed: %w", err)
		}

		if len(tpResp.Data) > 0 {
			tpAlgoId = tpResp.Data[0].AlgoId
			m.logger.Info("Take-Profit order placed successfully for %s (%s), algoId: %s, trigger: %.8f",
				position.Instrument, position.PositionSide, tpAlgoId, adjustedPrices.TpPrice)
		}
	} else {
		m.logger.Warn("Skipping Take-Profit order for %s (%s) due to price condition", position.Instrument, position.PositionSide)
	}

	// Place Stop-Loss order (if not skipped)
	if !skipSL {
		slReq := okx.AlgoOrderRequest{
			InstId:          position.Instrument,
			TdMode:          tdMode,
			Side:            orderSide,
			PosSide:         position.PositionSide.String(),
			OrdType:         "conditional",
			Sz:              formatFloat(size),
			SlTriggerPx:     formatFloat(adjustedPrices.SlPrice),
			SlOrdPx:         "-1", // Market order
			SlTriggerPxType: "last",
			ReduceOnly:      true,
		}

		m.logger.Debug("Placing Stop-Loss order for %s (%s): SL=%.8f", position.Instrument, position.PositionSide, adjustedPrices.SlPrice)

		slResp, err := m.okxClient.PlaceAlgoOrder(slReq)
		if err != nil {
			if tpAlgoId != "" {
				m.logger.Error("Stop-Loss order failed (TP order %s was placed): %v", tpAlgoId, err)
			}
			return fmt.Errorf("Stop-Loss order failed: %w", err)
		}

		if len(slResp.Data) > 0 {
			slAlgoId := slResp.Data[0].AlgoId
			m.logger.Info("Stop-Loss order placed successfully for %s (%s), algoId: %s, trigger: %.8f",
				position.Instrument, position.PositionSide, slAlgoId, adjustedPrices.SlPrice)
		}
	} else {
		m.logger.Error("Skipping Stop-Loss order for %s (%s) - CRITICAL: Manual intervention required!", position.Instrument, position.PositionSide)
	}

	if skipTP && skipSL {
		return fmt.Errorf("both TP and SL orders were skipped due to price conditions - manual intervention required")
	}

	m.logger.Info("TPSL orders placed successfully for %s (%s)", position.Instrument, position.PositionSide)
	return nil
}

// placeTPSLOrderOriginal 原始的下单逻辑（不验证当前价格）/ Original order placement logic without price validation
func (m *Manager) placeTPSLOrderOriginal(position *models.Position, size float64, prices *TPSLPrices) error {
	// This is the fallback method when we can't get current market price
	// Just place orders with calculated prices

	isLong := m.isLongPosition(position)
	var orderSide string
	if isLong {
		orderSide = "sell"
	} else {
		orderSide = "buy"
	}

	m.logger.Debug("Placing TPSL orders without price validation for %s (%s)", position.Instrument, position.PositionSide)

	// Determine trade mode from position
	tdMode := position.MarginMode.String()
	if tdMode == "" {
		tdMode = models.MarginModeCross.String() // Default to cross if not specified
	}

	// Place TP
	tpReq := okx.AlgoOrderRequest{
		InstId:          position.Instrument,
		TdMode:          tdMode,
		Side:            orderSide,
		PosSide:         position.PositionSide.String(),
		OrdType:         "conditional",
		Sz:              formatFloat(size),
		TpTriggerPx:     formatFloat(prices.TpPrice),
		TpOrdPx:         "-1",
		TpTriggerPxType: "last",
		ReduceOnly:      true,
	}

	tpResp, err := m.okxClient.PlaceAlgoOrder(tpReq)
	if err != nil {
		return fmt.Errorf("Take-Profit order failed: %w", err)
	}

	var tpAlgoId string
	if len(tpResp.Data) > 0 {
		tpAlgoId = tpResp.Data[0].AlgoId
		m.logger.Info("Take-Profit order placed for %s, algoId: %s", position.Instrument, tpAlgoId)
	}

	// Place SL
	slReq := okx.AlgoOrderRequest{
		InstId:          position.Instrument,
		TdMode:          tdMode,
		Side:            orderSide,
		PosSide:         position.PositionSide.String(),
		OrdType:         "conditional",
		Sz:              formatFloat(size),
		SlTriggerPx:     formatFloat(prices.SlPrice),
		SlOrdPx:         "-1",
		SlTriggerPxType: "last",
		ReduceOnly:      true,
	}

	slResp, err := m.okxClient.PlaceAlgoOrder(slReq)
	if err != nil {
		return fmt.Errorf("Stop-Loss order failed: %w", err)
	}

	if len(slResp.Data) > 0 {
		m.logger.Info("Stop-Loss order placed for %s, algoId: %s", position.Instrument, slResp.Data[0].AlgoId)
	}

	return nil
}
