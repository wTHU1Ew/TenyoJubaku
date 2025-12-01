package models

// MarginMode 保证金模式 / Margin mode type
type MarginMode string

const (
	// MarginModeCross 全仓模式 / Cross margin mode
	MarginModeCross MarginMode = "cross"

	// MarginModeIsolated 逐仓模式 / Isolated margin mode
	MarginModeIsolated MarginMode = "isolated"
)

// String 返回字符串表示 / Return string representation
func (m MarginMode) String() string {
	return string(m)
}

// IsValid 检查是否为有效的保证金模式 / Check if valid margin mode
func (m MarginMode) IsValid() bool {
	return m == MarginModeCross || m == MarginModeIsolated
}

// PositionSide 持仓方向 / Position side type
type PositionSide string

const (
	// PositionSideLong 多头持仓 / Long position
	PositionSideLong PositionSide = "long"

	// PositionSideShort 空头持仓 / Short position
	PositionSideShort PositionSide = "short"

	// PositionSideNet 单向持仓（净持仓模式）/ Net position (one-way mode)
	PositionSideNet PositionSide = "net"
)

// String 返回字符串表示 / Return string representation
func (p PositionSide) String() string {
	return string(p)
}

// IsValid 检查是否为有效的持仓方向 / Check if valid position side
func (p PositionSide) IsValid() bool {
	return p == PositionSideLong || p == PositionSideShort || p == PositionSideNet
}

// OrderSide 订单方向 / Order side type
type OrderSide string

const (
	// OrderSideBuy 买入 / Buy
	OrderSideBuy OrderSide = "buy"

	// OrderSideSell 卖出 / Sell
	OrderSideSell OrderSide = "sell"
)

// String 返回字符串表示 / Return string representation
func (o OrderSide) String() string {
	return string(o)
}

// IsValid 检查是否为有效的订单方向 / Check if valid order side
func (o OrderSide) IsValid() bool {
	return o == OrderSideBuy || o == OrderSideSell
}

// OrderType 订单类型 / Order type
type OrderType string

const (
	// OrderTypeConditional 条件单（止盈止损）/ Conditional order (TP/SL)
	OrderTypeConditional OrderType = "conditional"

	// OrderTypeMarket 市价单 / Market order
	OrderTypeMarket OrderType = "market"

	// OrderTypeLimit 限价单 / Limit order
	OrderTypeLimit OrderType = "limit"
)

// String 返回字符串表示 / Return string representation
func (o OrderType) String() string {
	return string(o)
}

// IsValid 检查是否为有效的订单类型 / Check if valid order type
func (o OrderType) IsValid() bool {
	return o == OrderTypeConditional || o == OrderTypeMarket || o == OrderTypeLimit
}

// TriggerPriceType 触发价格类型 / Trigger price type
type TriggerPriceType string

const (
	// TriggerPriceTypeLast 最新价 / Last traded price
	TriggerPriceTypeLast TriggerPriceType = "last"

	// TriggerPriceTypeIndex 指数价 / Index price
	TriggerPriceTypeIndex TriggerPriceType = "index"

	// TriggerPriceTypeMark 标记价 / Mark price
	TriggerPriceTypeMark TriggerPriceType = "mark"
)

// String 返回字符串表示 / Return string representation
func (t TriggerPriceType) String() string {
	return string(t)
}

// IsValid 检查是否为有效的触发价格类型 / Check if valid trigger price type
func (t TriggerPriceType) IsValid() bool {
	return t == TriggerPriceTypeLast || t == TriggerPriceTypeIndex || t == TriggerPriceTypeMark
}
