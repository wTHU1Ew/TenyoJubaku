# Database Query Guide

This document shows how to query your account balance and position data from the TenyoJubaku database.

**Note:** The system only records balances for **BTC, ETH, and USDT**. Other currencies are ignored.

## Prerequisites

Install SQLite3 if not already installed:
```bash
# macOS
brew install sqlite3

# Or check if already installed
sqlite3 --version
```

## Quick Start

Open the database:
```bash
sqlite3 data/tenyojubaku.db
```

## Common Queries

### 1. View Latest Account Balance (All Currencies)

```sql
-- Get the most recent balance for all currencies
SELECT
    datetime(timestamp) as time,
    currency,
    balance,
    available,
    frozen,
    equity
FROM account_balances
WHERE timestamp = (SELECT MAX(timestamp) FROM account_balances)
ORDER BY currency;
```

**Example Output:**
```
time                  currency  balance      available    frozen      equity
--------------------  --------  -----------  -----------  ----------  ----------
2025-11-28 03:30:00   USDT      1000.50      950.00       50.50       1000.50
2025-11-28 03:30:00   BTC       0.15000000   0.10000000   0.05000000  9500.00
```

### 2. View Balance for Specific Currency (e.g., USDT)

```sql
SELECT
    datetime(timestamp) as time,
    balance,
    available,
    frozen,
    equity
FROM account_balances
WHERE currency = 'USDT'
ORDER BY timestamp DESC
LIMIT 10;
```

### 3. View Current Open Positions

```sql
-- Get the most recent positions
SELECT
    datetime(timestamp) as time,
    instrument,
    position_side,
    position_size,
    average_price,
    unrealized_pnl,
    margin,
    leverage
FROM positions
WHERE timestamp = (SELECT MAX(timestamp) FROM positions)
ORDER BY instrument;
```

**Example Output:**
```
time                  instrument  position_side  position_size  average_price  unrealized_pnl  margin    leverage
--------------------  ----------  -------------  -------------  -------------  --------------  --------  --------
2025-11-28 03:30:00   BTC-USDT    long          1.5            50000.00       150.50          1000.00   5.0
2025-11-28 03:30:00   ETH-USDT    short         10.0           3000.00        -50.25          500.00    3.0
```

### 4. Calculate Total Account Value (in USD)

```sql
-- Sum all equity values
SELECT
    datetime(MAX(timestamp)) as latest_time,
    ROUND(SUM(equity), 2) as total_equity_usd
FROM account_balances
WHERE timestamp = (SELECT MAX(timestamp) FROM account_balances);
```

**Example Output:**
```
latest_time           total_equity_usd
--------------------  ----------------
2025-11-28 03:30:00   10500.50
```

### 5. View Balance History Over Time

```sql
-- USDT balance over the last 24 hours
SELECT
    datetime(timestamp) as time,
    balance,
    available,
    frozen
FROM account_balances
WHERE currency = 'USDT'
  AND timestamp >= datetime('now', '-1 day')
ORDER BY timestamp DESC;
```

### 6. Track Position PnL Over Time

```sql
-- BTC-USDT position PnL history
SELECT
    datetime(timestamp) as time,
    position_size,
    average_price,
    unrealized_pnl,
    ROUND((unrealized_pnl / margin * 100), 2) as pnl_percentage
FROM positions
WHERE instrument = 'BTC-USDT'
  AND timestamp >= datetime('now', '-1 day')
ORDER BY timestamp DESC
LIMIT 20;
```

### 7. Find Maximum and Minimum Balance

```sql
-- Highest and lowest USDT balance
SELECT
    'Highest' as type,
    datetime(timestamp) as time,
    balance
FROM account_balances
WHERE currency = 'USDT'
ORDER BY balance DESC
LIMIT 1

UNION ALL

SELECT
    'Lowest' as type,
    datetime(timestamp) as time,
    balance
FROM account_balances
WHERE currency = 'USDT'
ORDER BY balance ASC
LIMIT 1;
```

### 8. Count Data Points (How Many Records)

```sql
-- Check how many snapshots collected
SELECT
    COUNT(*) as total_balance_records,
    COUNT(DISTINCT currency) as currencies,
    datetime(MIN(timestamp)) as first_record,
    datetime(MAX(timestamp)) as last_record
FROM account_balances;
```

## Advanced Queries

### Calculate Balance Change Over Time

```sql
-- Compare current balance with 1 hour ago
WITH current AS (
    SELECT currency, balance as current_balance
    FROM account_balances
    WHERE timestamp = (SELECT MAX(timestamp) FROM account_balances)
),
previous AS (
    SELECT currency, balance as previous_balance
    FROM account_balances
    WHERE timestamp = (
        SELECT MAX(timestamp) FROM account_balances
        WHERE timestamp <= datetime('now', '-1 hour')
    )
)
SELECT
    c.currency,
    c.current_balance,
    p.previous_balance,
    ROUND(c.current_balance - p.previous_balance, 8) as change,
    ROUND((c.current_balance - p.previous_balance) / p.previous_balance * 100, 2) as change_percent
FROM current c
LEFT JOIN previous p ON c.currency = p.currency
WHERE p.previous_balance IS NOT NULL;
```

### Export Data to CSV

```sql
-- In sqlite3 command line, set output format
.mode csv
.headers on
.output balance_export.csv

SELECT
    datetime(timestamp) as time,
    currency,
    balance,
    available,
    frozen,
    equity
FROM account_balances
WHERE currency = 'USDT'
ORDER BY timestamp DESC;

.output stdout
```

## Useful SQLite Commands

```sql
-- Show all tables
.tables

-- Show table schema
.schema account_balances
.schema positions

-- Enable column headers
.headers on

-- Better formatting
.mode column

-- Show query execution time
.timer on

-- Exit sqlite3
.quit
```

## Real-Time Monitoring Script

Create a shell script to monitor your balance:

```bash
#!/bin/bash
# monitor_balance.sh

while true; do
    clear
    echo "=== TenyoJubaku Account Monitor ==="
    echo "Updated: $(date)"
    echo ""

    sqlite3 data/tenyojubaku.db <<EOF
.mode column
.headers on
SELECT
    datetime(timestamp) as 'Time',
    currency as 'Currency',
    ROUND(balance, 2) as 'Balance',
    ROUND(available, 2) as 'Available',
    ROUND(frozen, 2) as 'Frozen'
FROM account_balances
WHERE timestamp = (SELECT MAX(timestamp) FROM account_balances)
ORDER BY currency;
EOF

    sleep 60  # Refresh every 60 seconds
done
```

Make it executable and run:
```bash
chmod +x monitor_balance.sh
./monitor_balance.sh
```

## Python Script for Analysis

```python
# analyze_balance.py
import sqlite3
import pandas as pd
from datetime import datetime, timedelta

# Connect to database
conn = sqlite3.connect('data/tenyojubaku.db')

# Query last 24 hours of USDT balance
query = """
SELECT
    timestamp,
    balance,
    available,
    frozen
FROM account_balances
WHERE currency = 'USDT'
  AND timestamp >= datetime('now', '-1 day')
ORDER BY timestamp
"""

df = pd.read_sql_query(query, conn)
df['timestamp'] = pd.to_datetime(df['timestamp'])

# Calculate statistics
print("USDT Balance Statistics (Last 24 Hours)")
print("=" * 50)
print(f"Current Balance: ${df['balance'].iloc[-1]:.2f}")
print(f"Average Balance: ${df['balance'].mean():.2f}")
print(f"Max Balance:     ${df['balance'].max():.2f}")
print(f"Min Balance:     ${df['balance'].min():.2f}")
print(f"Change:          ${df['balance'].iloc[-1] - df['balance'].iloc[0]:.2f}")

# Close connection
conn.close()
```

## Backup Your Data

```bash
# Create backup
cp data/tenyojubaku.db backups/tenyojubaku_$(date +%Y%m%d_%H%M%S).db

# Or use SQLite backup command
sqlite3 data/tenyojubaku.db ".backup backups/backup_$(date +%Y%m%d).db"
```

## Troubleshooting

### Database is locked
If you get "database is locked" error:
```bash
# Check if TenyoJubaku is running
ps aux | grep tenyojubaku

# Wait for write to complete, or stop the service temporarily
```

### No data found
```sql
-- Check if any data exists
SELECT COUNT(*) FROM account_balances;
SELECT COUNT(*) FROM positions;

-- Check timestamps
SELECT MIN(timestamp), MAX(timestamp) FROM account_balances;
```

### Time zone confusion
All timestamps are stored in UTC. Convert to your local time:
```sql
-- Convert UTC to local time (example for UTC+8)
SELECT
    datetime(timestamp, '+8 hours') as local_time,
    currency,
    balance
FROM account_balances
WHERE timestamp = (SELECT MAX(timestamp) FROM account_balances);
```
