package tpsl

import (
	"fmt"
	"math"
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

		// Place TPSL order
		err = m.placeTPSLOrder(position, uncoveredSize, prices)
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
// Parameters:
//   - position: 持仓信息 / Position information
//   - algoOrders: 算法订单列表 / List of algo orders
//
// Returns:
//   - float64: 未覆盖的持仓大小 / Uncovered position size
func (m *Manager) analyzeCoverage(position *models.Position, algoOrders []okx.AlgoOrder) float64 {
	coveredSize := 0.0

	// Filter and sum matching algo orders
	for _, order := range algoOrders {
		if m.matchesPosition(&order, position) {
			// Parse order size
			size, err := strconv.ParseFloat(order.Sz, 64)
			if err != nil {
				m.logger.Warn("Failed to parse algo order size '%s' for order %s: %v", order.Sz, order.AlgoId, err)
				continue
			}
			coveredSize += size
			m.logger.Debug("Found matching algo order %s with size %.8f for position %s",
				order.AlgoId, size, position.Instrument)
		}
	}

	uncoveredSize := position.PositionSize - coveredSize
	if uncoveredSize < 0 {
		uncoveredSize = 0 // Shouldn't happen, but handle gracefully
	}

	m.logger.Debug("Position %s coverage: total=%.8f, covered=%.8f, uncovered=%.8f",
		position.Instrument, position.PositionSize, coveredSize, uncoveredSize)

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
	if order.PosSide != position.PositionSide {
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
	if position.PositionSide == "long" {
		return true
	}
	if position.PositionSide == "short" {
		return false
	}

	// In one-way (net) mode, determine from position size
	// Positive size = long, negative size = short
	return position.PositionSize > 0
}

// placeTPSLOrder 下单TPSL订单 / Place TPSL order
// 为持仓下单止盈止损订单
// Place take-profit and stop-loss order for position
//
// Parameters:
//   - position: 持仓信息 / Position information
//   - size: 订单大小 / Order size
//   - prices: TPSL价格 / TPSL prices
//
// Returns:
//   - error: 下单失败时返回错误 / Error on placement failure
func (m *Manager) placeTPSLOrder(position *models.Position, size float64, prices *TPSLPrices) error {
	// Determine order side (opposite of position)
	isLong := m.isLongPosition(position)
	var orderSide string
	if isLong {
		orderSide = "sell" // Close long position
	} else {
		orderSide = "buy" // Close short position
	}

	// Build algo order request
	req := okx.AlgoOrderRequest{
		InstId:        position.Instrument,
		TdMode:        "cross", // Will be enhanced in Task 3.3 to read from position
		Side:          orderSide,
		PosSide:       position.PositionSide,
		OrdType:       "conditional",
		Sz:            formatFloat(size),
		TpTriggerPx:   formatFloat(prices.TpPrice),
		TpOrdPx:       "-1", // Market order
		SlTriggerPx:   formatFloat(prices.SlPrice),
		SlOrdPx:       "-1", // Market order
		ReduceOnly:    true,
		TpTriggerPxType: "last",
		SlTriggerPxType: "last",
	}

	m.logger.Debug("Placing TPSL order for %s (%s): side=%s, size=%.8f, TP=%.8f, SL=%.8f",
		position.Instrument, position.PositionSide, orderSide, size, prices.TpPrice, prices.SlPrice)

	// Place order
	resp, err := m.okxClient.PlaceAlgoOrder(req)
	if err != nil {
		return fmt.Errorf("API call failed: %w", err)
	}

	// Extract algo ID
	if len(resp.Data) > 0 {
		algoId := resp.Data[0].AlgoId
		m.logger.Info("TPSL order placed successfully for %s (%s), algoId: %s",
			position.Instrument, position.PositionSide, algoId)
	} else {
		m.logger.Info("TPSL order placed successfully for %s (%s)",
			position.Instrument, position.PositionSide)
	}

	return nil
}

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

// roundToDecimal 四舍五入到指定小数位 / Round to specified decimal places
func roundToDecimal(f float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(f*pow) / pow
}
