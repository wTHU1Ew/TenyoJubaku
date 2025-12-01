# Proposal: Add Order Frequency Control

**Change ID:** add-order-frequency-control
**Status:** Draft
**Created:** 2025-12-01
**Author:** Claude Code

## Overview

This proposal introduces comprehensive order frequency control and maker-only trading restrictions to the TenyoJubaku trading system. The goal is to minimize drawdowns and achieve stable compounding by enforcing strict trading discipline through automated controls.

## Goals

1. **Order Frequency Limit**: Restrict the number of orders placed per week to prevent overtrading (default: max 5 orders/week, configurable)
2. **Maker-Only Trading**: Enforce maker-only order placement to avoid FOMO and ensure better price execution (taker allowed only for partial TP/SL with max 50% default)
3. **Price Distance Validation**: Ensure limit orders are placed at least 1% away from current market price (configurable) to avoid impulsive trades
4. **Multi-Confirmation System**: Implement a confirmation and timeout mechanism where pending orders require periodic confirmation (every 12 hours default) with 4-hour waiting period, automatically reducing order size to 50% on timeout

## Background

As described in Feature 3 of the project requirements, the system needs strict trading restrictions to prevent emotional trading decisions and enforce disciplined trading practices. Current implementation (Features 1 & 2) provides monitoring and automatic TPSL management, but lacks controls on new order placement.

## Scope

This proposal covers:
- Order frequency tracking and enforcement
- Maker vs taker distinction and validation
- Price distance validation for limit orders
- Multi-step confirmation workflow for pending orders
- Timeout handling with automatic order size reduction
- Configuration management for all control parameters

This proposal does NOT cover:
- Order entry notes (Feature 5)
- Planned trading for extreme markets (Feature 4)
- On-chain data acquisition (Feature 6)

## Design Highlights

The implementation will introduce three main capabilities:

### 1. Order Frequency Limit
- Track order placements in a rolling weekly window
- Reject new orders when weekly limit is exceeded
- Persist order history in database for accurate counting
- Configurable weekly limit (default: 5)

### 2. Maker-Only Trading
- Validate all new orders are limit orders (maker)
- Exception: allow market/taker orders for TP/SL closing positions (reduce-only)
- Enforce maximum taker percentage for partial closes (default: 50%)
- Validate price distance from market price (default: 1% minimum)
- Query current market price from OKX ticker API

### 3. Order Confirmation System
- Maintain pending order state in database
- Send periodic confirmation notifications (default: every 12 hours)
- Track confirmation status with timeout logic (default: 4-hour waiting period)
- Automatically amend order size to 50% (configurable) on timeout
- Cancel orders after multiple failed confirmations (configurable)

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| False rejection of valid orders | High - prevents legitimate trades | Comprehensive validation logic, clear error messages, manual override capability via config |
| Database corruption affecting order count | Medium - incorrect frequency tracking | Use transactions, add data integrity checks, recovery from OKX order history |
| Notification system failure | Medium - no confirmation prompts | Log all confirmation requests, provide CLI command to check pending confirmations |
| Market price API failure | Medium - cannot validate price distance | Retry logic with backoff, fallback to last known price with staleness check |
| Timezone confusion in weekly windows | Low - incorrect frequency counting | Use UTC consistently, document clearly in logs and config |

## Dependencies

- Existing monitoring system (Feature 1) for position tracking
- Existing TPSL management system (Feature 2) for TP/SL order validation
- OKX API endpoints:
  - `POST /api/v5/trade/order` - Place order
  - `GET /api/v5/market/ticker` - Get current market price
  - `POST /api/v5/trade/amend-order` - Amend pending order size
  - `POST /api/v5/trade/cancel-order` - Cancel order
  - `GET /api/v5/trade/orders-pending` - Get pending orders
- SQLite database for order history and confirmation state

## Success Criteria

1. System correctly enforces weekly order frequency limit across restarts
2. System rejects taker orders except for valid TP/SL scenarios
3. System validates price distance meets minimum threshold before placement
4. Pending orders trigger confirmation notifications at configured intervals
5. Orders automatically reduce size on timeout as configured
6. All configuration parameters are properly loaded and validated
7. Comprehensive logging for all enforcement actions and rejections
8. Unit tests achieve >90% coverage for validation logic
9. Integration tests verify end-to-end order placement workflow with all controls

## Open Questions

None - requirements are clearly defined in project.md Feature 3.

## Related Changes

- Builds on: `add-account-monitoring` (Feature 1)
- Builds on: `add-auto-tpsl-management` (Feature 2)
- Future: Will be complemented by order entry notes (Feature 5)
