# Proposal: Automatic Stop-Loss and Take-Profit Management

## Change ID
`add-auto-tpsl-management`

## Overview
Implement automatic stop-loss (SL) and take-profit (TP) order management to minimize drawdowns and enforce disciplined risk management. The system will monitor all open positions and automatically add or complete TPSL orders when positions lack proper protection or are only partially protected.

## Problem Statement
Manual trading often leads to positions being opened without proper stop-loss and take-profit orders, exposing the account to excessive risk. Even when TPSL orders exist, they may not cover the entire position, leaving partial exposure. This creates the following issues:

1. **Risk Exposure**: Positions without stop-loss orders can lead to catastrophic losses
2. **Inconsistent Risk Management**: Manual TPSL placement leads to inconsistent risk-reward ratios
3. **Emotional Decision Making**: Lack of automated TPSL allows emotional interference during trading
4. **Partial Coverage**: Existing TPSL orders may only cover partial positions, leaving gaps in risk management

## Proposed Solution
Implement an automated TPSL management system that:

1. **Position Monitoring**: Continuously monitor all open positions to identify those without TPSL coverage
2. **Automatic TPSL Placement**: Automatically place stop-loss and take-profit orders for positions without protection
3. **Partial Coverage Completion**: Add additional TPSL orders when existing orders don't cover the full position size
4. **Risk-Based Calculation**: Calculate TPSL prices based on configurable risk parameters (volatility percentage, profit-loss ratio, leverage)
5. **OKX API Integration**: Use OKX algo order API to place conditional TPSL orders

## Key Features

### 1. TPSL Detection and Analysis
- Fetch pending algo orders (conditional orders) from OKX API
- Compare algo order coverage against actual position sizes
- Identify positions that are:
  - Completely unprotected (no TPSL orders)
  - Partially protected (TPSL covers less than 100% of position)

### 2. TPSL Calculation Engine
- **Stop-Loss Calculation**:
  - Base: 1% of entry price (configurable volatility percentage)
  - Adjusted for leverage (e.g., 5x leverage = 5% position loss at 1% price move)
  - Formula: `SL_price = entry_price ± (entry_price × volatility% × leverage_factor)`

- **Take-Profit Calculation**:
  - Based on configured profit-loss ratio (default 5:1)
  - Formula: `TP_price = entry_price ± (entry_price × volatility% × leverage_factor × pl_ratio)`

### 3. Automatic Order Placement
- Place conditional algo orders via OKX `/api/v5/trade/order-algo` endpoint
- Set `ordType` = `conditional` for TPSL orders
- Set `reduceOnly` = `true` to ensure orders only close positions
- Use market orders (`orderPx` = `-1`) for immediate execution when triggered
- Calculate order sizes to cover uncovered position portions

### 4. Separate TPSL Scheduler
- Independent scheduler running at configurable intervals (default: 5 minutes)
- Separate from the monitoring loop to avoid coupling
- Configurable enable/disable via config file
- Graceful error handling with logging

## Architecture Decisions

### Separation of Concerns
- **Monitoring Module**: Continues to fetch and store position/balance data
- **TPSL Module**: New independent module responsible for TPSL management
- Clean interfaces between modules via shared models

### Configuration-Driven
All risk parameters configurable via YAML:
- Volatility percentage (default: 1%)
- Profit-loss ratio (default: 5:1)
- TPSL check interval (default: 300s)
- Enable/disable flag

### Additive Approach
When positions have partial TPSL coverage, add NEW orders rather than amending existing ones. This:
- Avoids canceling manually placed orders
- Preserves user's existing TPSL strategy
- Minimizes API call complexity
- Reduces risk of accidentally removing protection

### Simple Volatility Calculation
Use simple percentage from entry price rather than complex ATR/standard deviation calculations:
- No need for historical candlestick data fetching
- Simpler, more predictable calculations
- Faster execution
- Easier to understand and debug

## Dependencies
- Existing account monitoring system (positions data)
- OKX Trade API (algo order placement and querying)
- Configuration management (new TPSL settings)
- Logging infrastructure

## Risks and Mitigations

### Risk 1: API Rate Limits
- **Mitigation**: Run TPSL checks at longer intervals (5 minutes default)
- **Mitigation**: Implement exponential backoff on rate limit errors
- **Mitigation**: Batch operations when possible

### Risk 2: Incorrect TPSL Price Calculation
- **Mitigation**: Comprehensive unit tests for calculation logic
- **Mitigation**: Detailed logging of all calculations
- **Mitigation**: Configuration validation on startup

### Risk 3: Double-Placement of Orders
- **Mitigation**: Check existing algo orders before placing new ones
- **Mitigation**: Track recently placed orders to avoid duplicates
- **Mitigation**: Use algo order IDs to detect existing coverage

### Risk 4: Order Placement Failures
- **Mitigation**: Retry logic with exponential backoff
- **Mitigation**: Detailed error logging
- **Mitigation**: Continue monitoring even if placement fails

## Success Criteria
1. System automatically places TPSL orders for all unprotected positions within one check interval
2. TPSL calculations correctly account for leverage and configured risk parameters
3. Partial positions are properly covered by additional TPSL orders
4. No duplicate TPSL orders are created for the same position coverage
5. All TPSL operations are logged with sufficient detail for auditing
6. System handles API failures gracefully without crashing

## Out of Scope
- Manual override/exclusion of specific positions (all positions get auto-TPSL)
- Amending existing TPSL orders (only additive approach)
- Dynamic adjustment of TPSL based on market conditions
- Multiple TPSL strategies per position
- Trailing stop-loss functionality
- ATR or complex volatility calculations

## Related Capabilities
This change introduces one new capability:
- **auto-tpsl-management**: Automatic stop-loss and take-profit order management

## Timeline
This is a planning proposal. Implementation will be tracked via tasks.md.
