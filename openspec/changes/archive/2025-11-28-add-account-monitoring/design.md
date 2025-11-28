# Design Document: Real-time Account and Position Monitoring

## Context
The TenyoJubaku trading system needs to continuously monitor OKX trading account balances and positions to enable automated risk management features. The monitoring must be reliable, secure, and support future migration from macOS to NAS deployment.

**Constraints:**
- Runs on macOS 14 initially, must support future NAS migration
- Uses OKX API (www.okx.com) for account data
- Must store data frequently (approximately every minute)
- No sensitive data (credentials, logs) in remote repository
- Bilingual code comments (Chinese function briefs + English details)
- Following project conventions: CamelCase, OOP principles, layered architecture

**Stakeholders:**
- Solo developer/trader using the system
- Future maintainers (post-NAS migration)

## Goals / Non-Goals

**Goals:**
- Fetch account balance and position data from OKX API every ~60 seconds
- Persist data to local database for historical analysis
- Log connection status, API responses, and errors to file
- Support configuration without exposing credentials to repository
- Provide foundation for future features (stop-loss automation, order frequency limits)

**Non-Goals:**
- Real-time WebSocket streaming (polling is sufficient for ~1min intervals)
- Multi-exchange support (OKX only for now)
- Web dashboard or UI (future feature)
- Historical data backfill (start fresh from deployment)
- Advanced analytics or alerting (future features)

## Decisions

### 1. Polling vs WebSocket
**Decision:** Use REST API polling every 60 seconds

**Rationale:**
- Simpler implementation and error recovery
- Sufficient for ~1 minute monitoring interval requirement
- Lower complexity for initial version
- REST API is more stable for long-running processes
- Can upgrade to WebSocket later if needed

**Alternatives considered:**
- WebSocket (`Account channel`, `Positions channel`): More complex, requires persistent connection management, overkill for 60s intervals

### 2. Database Selection
**Decision:** Use SQLite

**Rationale:**
- Lightweight, no separate database server required
- Embedded database simplifies deployment
- Sufficient for single-user, frequent writes (~1 write/minute)
- Easy to backup and migrate (single file)
- Good performance for time-series queries

**Alternatives considered:**
- PostgreSQL/MySQL: Over-engineered for single-user system, requires separate server
- InfluxDB: Optimized for time-series but adds complexity and deployment overhead

### 3. Layered Architecture
**Decision:** Three-layer architecture (API Client → Service → Storage)

**Structure:**
```
┌─────────────────┐
│  Main App       │  - Initialization, configuration, signal handling
└────────┬────────┘
         │
┌────────▼────────┐
│  Monitor Service│  - Scheduling, orchestration, error recovery
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
┌───▼───┐ ┌──▼──────┐
│ OKX   │ │ Storage │  - API Client: OKX communication, auth
│ Client│ │ Layer   │  - Storage: Database operations
└───────┘ └─────────┘
```

**Rationale:**
- Clear separation of concerns
- Testable components (can mock API client in tests)
- Follows project requirement for layered design
- Easy to extend (add WebSocket support later, swap database)

### 4. Configuration Management
**Decision:** YAML configuration file with template approach

**Structure:**
- `configs/config.template.yaml` - Committed to repository with placeholder values
- `configs/config.yaml` - User's actual config (in .gitignore)
- Environment variables as fallback for sensitive data

**Rationale:**
- Clear separation between template and actual credentials
- Meets requirement: no sensitive data in repository
- YAML is human-readable and supports comments
- Template file serves as documentation

### 5. Logging Strategy
**Decision:** Structured logging to rotating log files

**Requirements:**
- Log levels: DEBUG, INFO, WARN, ERROR
- Rotate daily or by size (e.g., 100MB)
- Connection status logged at startup and on failures
- API requests/responses logged (with sensitive data masked)
- All logs in `/logs` directory (excluded from git)

**Rationale:**
- File logging requirement explicitly mentioned by user
- Rotation prevents unbounded disk usage
- Structured logging aids debugging
- Sensitive data masking meets security requirements

### 6. Error Handling and Recovery
**Decision:** Retry with exponential backoff, graceful degradation

**Strategy:**
- API failures: Retry up to 3 times with exponential backoff (1s, 2s, 4s)
- Connection failures: Log error, wait for next polling interval
- Database failures: Critical error, log and exit (requires manual intervention)
- Graceful shutdown on SIGINT/SIGTERM

**Rationale:**
- Transient network errors shouldn't crash the system
- Database failures are critical (data loss risk)
- Explicit shutdown handling required for long-running service

### 7. Data Model Design
**Decision:** Separate tables for balances and positions with timestamp-based queries

**Schema:**
```sql
CREATE TABLE account_balances (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    currency VARCHAR(10) NOT NULL,
    balance DECIMAL(20,8) NOT NULL,
    available DECIMAL(20,8) NOT NULL,
    frozen DECIMAL(20,8) NOT NULL,
    equity DECIMAL(20,8),
    INDEX idx_timestamp (timestamp)
);

CREATE TABLE positions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME NOT NULL,
    instrument VARCHAR(50) NOT NULL,
    position_side VARCHAR(10) NOT NULL,  -- 'long' or 'short'
    position_size DECIMAL(20,8) NOT NULL,
    average_price DECIMAL(20,8) NOT NULL,
    unrealized_pnl DECIMAL(20,8) NOT NULL,
    margin DECIMAL(20,8) NOT NULL,
    leverage DECIMAL(5,2),
    INDEX idx_timestamp_instrument (timestamp, instrument)
);
```

**Rationale:**
- Normalized design (separate concerns)
- Timestamp indexing for fast time-range queries
- Sufficient precision for crypto amounts (DECIMAL 20,8)
- Supports future queries for analysis and reporting

### 8. OKX API Integration
**Decision:** Use OKX REST API v5 with signature-based authentication

**Endpoints:**
- `GET /api/v5/account/balance` - Account balance
- `GET /api/v5/account/positions` - Open positions

**Authentication:**
- API Key + Secret + Passphrase (stored in config)
- Signature: Base64(HMAC-SHA256(timestamp + method + path + body, secret))
- Headers: `OK-ACCESS-KEY`, `OK-ACCESS-SIGN`, `OK-ACCESS-TIMESTAMP`, `OK-ACCESS-PASSPHRASE`

**Rate Limiting:**
- Respect OKX rate limits (documented in API docs)
- Built-in delay between requests
- Exponential backoff on rate limit errors

**Rationale:**
- REST API is well-documented and stable
- Signature auth is OKX's standard authentication method
- Rate limiting prevents account suspension

## Risks / Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| API credential exposure | **HIGH** - Unauthorized access to trading account | Store credentials only in local config (gitignored), add validation checks, document security best practices in README |
| Database corruption | **MEDIUM** - Loss of historical data | Regular backups (document in deployment guide), WAL mode for SQLite |
| OKX API changes | **MEDIUM** - System breaks on API updates | Version API endpoints explicitly, add logging for unexpected responses, monitor OKX changelog |
| Network failures | **LOW** - Missed data points | Retry logic, log failures, acceptable to skip intervals |
| Clock drift | **LOW** - Timestamp accuracy issues | Use system time, document NTP requirement for NAS deployment |

**Trade-offs:**
- Polling (simplicity) vs WebSocket (real-time) → Chose simplicity for initial version
- SQLite (lightweight) vs PostgreSQL (scalable) → Chose lightweight for single-user deployment
- File logging (simple) vs Centralized logging (advanced) → Chose file logging per requirement

## Migration Plan

### Initial Deployment (macOS)
1. Install Go and SQLite
2. Clone repository
3. Copy `config.template.yaml` to `config.yaml` and fill in OKX credentials
4. Run application: `go run cmd/main.go`

### Future NAS Migration
1. Build for NAS architecture (likely Linux ARM/x86)
2. Copy binary, config, and database file to NAS
3. Set up systemd service or supervisor for auto-restart
4. Configure log rotation via logrotate
5. Set up automated backups of SQLite database

**Compatibility considerations:**
- Use standard Go libraries (avoid macOS-specific dependencies)
- Document cross-compilation process
- Test on Linux before NAS deployment

## Open Questions
1. **Q:** Should we store every polling snapshot, or only when values change?
   - **A:** Store every snapshot for accurate historical analysis (storage is cheap, data is valuable)

2. **Q:** What happens if position information is empty (no open positions)?
   - **A:** Log "no positions" at INFO level, don't insert empty records (avoid database bloat)

3. **Q:** Should we add a health check endpoint for future monitoring?
   - **A:** Not in initial version (no web interface yet), can add later when needed

4. **Q:** How to handle timezone differences between local system and OKX API?
   - **A:** Store all timestamps in UTC, convert to local time only for display/logging
