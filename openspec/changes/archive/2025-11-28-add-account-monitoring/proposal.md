# Change: Add Real-time Account and Position Monitoring

## Why
The TenyoJubaku trading system requires continuous monitoring of account funds and position information to enforce trading restrictions and enable automated risk management features like stop-loss/take-profit automation and order frequency limits. Without real-time account data, the system cannot make informed decisions about position sizing, risk exposure, or trigger automated protective measures.

## What Changes
- Add OKX API integration layer for authentication and account data retrieval
- Implement real-time monitoring service that polls account balance and positions approximately every minute
- Create database schema to persist account balance and position snapshots
- Add logging infrastructure for connection status, API responses, and system events
- Implement data models for account balance and position information
- Create configuration system for API credentials and monitoring parameters

## Impact
- Affected specs:
  - `account-monitoring` (new capability)
- Affected code:
  - New `/internal/okx` package for OKX API client
  - New `/internal/monitor` package for monitoring service
  - New `/internal/storage` package for database operations
  - New `/pkg/models` package for data structures
  - New `/configs` directory for configuration templates
  - New `/logs` directory for log files (gitignored)
- External dependencies:
  - OKX REST API (GET /api/v5/account/balance, GET /api/v5/account/positions)
  - Database (SQLite preferred for lightweight deployment)
  - Configuration file (must not contain actual credentials in repository)
