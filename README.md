# TenyoJubaku (天與咒縛)

A trading system with strict trading restrictions designed to minimize drawdowns and achieve stable compounding.

## Overview

TenyoJubaku is an automated trading system for OKX Exchange that implements comprehensive risk management through monitoring, automatic stop-loss/take-profit, and order frequency controls.

## Features

### Phase 1 (Current - 2025)
- **Real-time Account Monitoring**: Continuous monitoring of trading account funds and position information with data persistence
- **Automatic Stop-Loss/Take-Profit**: Auto-completion of protective orders based on volatility and profit-loss ratios
- **Order Frequency Limits**: Maker-only restrictions with confirmation workflows to prevent FOMO trading

### Phase 2 (Future)
- Planned trading for extreme market conditions
- Order entry notes with AI-powered market summaries
- On-chain data acquisition and analysis

## Architecture

TenyoJubaku follows a layered architecture:

```
┌─────────────────┐
│  Main App       │  Configuration, signal handling
└────────┬────────┘
         │
┌────────▼────────┐
│  Monitor Service│  Scheduling, orchestration
└────────┬────────┘
         │
    ┌────┴────┐
    │         │
┌───▼───┐ ┌──▼──────┐
│ OKX   │ │ Storage │
│ Client│ │ Layer   │
└───────┘ └─────────┘
```

## Prerequisites

- Go 1.21 or higher
- SQLite3
- OKX Exchange account with API credentials

## Installation

1. Clone the repository:
```bash
git clone git@github.com:wTHU1Ew/TenyoJubaku.git
cd TenyoJubaku
```

2. Install dependencies:
```bash
go mod download
```

3. Configure the application:
```bash
cp configs/config.template.yaml configs/config.yaml
# Edit configs/config.yaml with your OKX API credentials
```

## Configuration

⚠️ **SECURITY WARNING**: Never commit `configs/config.yaml` to version control. It contains sensitive API credentials.

Edit `configs/config.yaml` and provide:
- `okx.api_key`: Your OKX API key
- `okx.api_secret`: Your OKX API secret
- `okx.passphrase`: Your OKX API passphrase
- `monitoring.interval`: Monitoring interval in seconds (default: 60)
- `database.path`: Path to SQLite database file
- `logging.file_path`: Path to log file
- `logging.level`: Log level (DEBUG, INFO, WARN, ERROR)

### API Key Setup

1. Log in to your OKX account
2. Navigate to API Management
3. Create a new API key with **Read** permissions (trading permissions not required for monitoring)
4. Save your API key, secret, and passphrase securely
5. Add them to `configs/config.yaml`

## Usage

Run the monitoring service:

```bash
go run cmd/main.go
```

Or build and run:

```bash
go build -o bin/tenyojubaku cmd/main.go
./bin/tenyojubaku
```

The service will:
1. Validate configuration and API credentials
2. Initialize the database
3. Perform a health check
4. Start continuous monitoring (fetching account data every ~60 seconds)
5. Log all operations to `logs/app.log`

Press `Ctrl+C` to gracefully shut down the service.

## Database

Account balances and positions are stored in SQLite at `data/tenyojubaku.db`.

Tables:
- `account_balances`: Account balance snapshots (timestamp, currency, balance, available, frozen, equity)
  - **Only records BTC, ETH, and USDT** (other currencies are ignored)
- `positions`: Position snapshots (timestamp, instrument, side, size, avg_price, unrealized_pnl, margin, leverage)

All timestamps are stored in UTC.

## Logging

Logs are written to `logs/app.log` with automatic rotation:
- Maximum file size: 100MB
- Retention: 30 days
- Sensitive data (API keys, secrets) are automatically masked in logs

Log levels:
- **DEBUG**: Detailed API request/response information
- **INFO**: Normal operations (monitoring cycles, connection status)
- **WARN**: Recoverable errors (retry attempts, rate limits)
- **ERROR**: Critical errors (authentication failures, database errors)

## Testing

Run unit tests:
```bash
go test ./...
```

Run integration tests:
```bash
go test -tags=integration ./...
```

⚠️ **Testing Note**: End-to-end tests with real API calls use a maximum of 5 USDT for order placement tests.

## Development

Project structure:
```
TenyoJubaku/
├── cmd/                    # Main application entry point
├── internal/
│   ├── config/            # Configuration management
│   ├── logger/            # Logging infrastructure
│   ├── okx/               # OKX API client
│   ├── monitor/           # Monitoring service
│   └── storage/           # Database layer
├── pkg/
│   └── models/            # Data models
├── configs/               # Configuration files
│   └── config.template.yaml
├── data/                  # Database files (gitignored)
├── logs/                  # Log files (gitignored)
└── README.md
```

### Code Conventions

- **Naming**: CamelCase (variables start lowercase, functions start uppercase)
- **Comments**: Bilingual function briefs (Chinese summary + English details)
- **Architecture**: OOP principles with loose coupling
- **Testing**: Unit tests for all layers, integration tests for workflows

## Deployment

### macOS (Current)
Run directly with `go run` or build a binary.

### NAS (Future)
1. Cross-compile for your NAS architecture:
```bash
GOOS=linux GOARCH=amd64 go build -o bin/tenyojubaku-linux cmd/main.go
```

2. Transfer binary, config, and database to NAS

3. Set up systemd service or supervisor for auto-restart

4. Configure log rotation with logrotate

5. Set up automated database backups

## Security

- ✅ API credentials stored only in local config (gitignored)
- ✅ Sensitive data masked in all logs
- ✅ Read-only API permissions required (no trading permissions needed)
- ✅ Database and logs excluded from version control
- ⚠️ Keep your `config.yaml` secure and never share it
- ⚠️ Use API keys with minimal required permissions
- ⚠️ Regularly rotate API credentials

## Support

For issues or questions:
1. Check logs in `logs/app.log`
2. Verify API credentials and permissions
3. Ensure network connectivity to OKX API
4. Review OKX API status at https://www.okx.com/docs-v5/en/

## License

Private project - All rights reserved

## Disclaimer

This software is for personal use only. Cryptocurrency trading carries significant risk. The authors are not responsible for any financial losses incurred while using this software. Use at your own risk.
