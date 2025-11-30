package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

// Storage 数据库存储层 / Database storage layer
type Storage struct {
	db *sql.DB
}

// New 创建新的存储实例 / Create new storage instance
// 初始化SQLite数据库连接，创建表结构，配置连接池
// Initialize SQLite database connection, create table schema, configure connection pool
//
// Parameters:
//   - dbPath: Database file path (e.g., "./data/tenyojubaku.db"), directory will be created if not exists
//   - walMode: Whether to enable WAL (Write-Ahead Logging) mode for better concurrency performance
//   - maxOpenConns: Maximum number of open connections
//   - maxIdleConns: Maximum number of idle connections
//
// Returns:
//   - *Storage: 已初始化的存储实例，包含数据库连接和表结构
//     Initialized storage instance with database connection and table schema
//   - error: 数据库创建失败或表结构初始化失败时返回错误
//     Error on database creation failure or schema initialization failure
func New(dbPath string, walMode bool, maxOpenConns, maxIdleConns int) (*Storage, error) {
	// Ensure database directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)

	// Enable WAL mode if requested
	if walMode {
		if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
		}
	}

	storage := &Storage{db: db}

	// Initialize database schema
	if err := storage.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema 初始化数据库架构 / Initialize database schema
func (s *Storage) initSchema() error {
	// Create account_balances table
	accountBalancesSchema := `
	CREATE TABLE IF NOT EXISTS account_balances (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		currency VARCHAR(10) NOT NULL,
		balance REAL NOT NULL,
		available REAL NOT NULL,
		frozen REAL NOT NULL,
		equity REAL
	);
	CREATE INDEX IF NOT EXISTS idx_account_balances_timestamp ON account_balances(timestamp);
	CREATE INDEX IF NOT EXISTS idx_account_balances_currency ON account_balances(currency);
	`

	if _, err := s.db.Exec(accountBalancesSchema); err != nil {
		return fmt.Errorf("failed to create account_balances table: %w", err)
	}

	// Create positions table
	positionsSchema := `
	CREATE TABLE IF NOT EXISTS positions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		instrument VARCHAR(50) NOT NULL,
		position_side VARCHAR(10) NOT NULL,
		position_size REAL NOT NULL,
		average_price REAL NOT NULL,
		unrealized_pnl REAL NOT NULL,
		margin REAL NOT NULL,
		leverage REAL
	);
	CREATE INDEX IF NOT EXISTS idx_positions_timestamp ON positions(timestamp);
	CREATE INDEX IF NOT EXISTS idx_positions_timestamp_instrument ON positions(timestamp, instrument);
	`

	if _, err := s.db.Exec(positionsSchema); err != nil {
		return fmt.Errorf("failed to create positions table: %w", err)
	}

	return nil
}

// InsertAccountBalance 插入账户余额记录 / Insert account balance record
// 将账户余额数据写入account_balances表，记录时间戳和币种余额信息
// Write account balance data to account_balances table with timestamp and currency balance info
//
// Parameters:
//   - balance: Account balance data model, must contain valid Currency, Balance, Available fields
//     Timestamp will be converted to UTC for storage
//
// Returns:
//   - error: 数据验证失败或数据库写入失败时返回错误 / Error on validation failure or database write failure
//     成功时会将生成的ID回写到balance.ID字段 / On success, generated ID is written back to balance.ID
func (s *Storage) InsertAccountBalance(balance *models.AccountBalance) error {
	if err := balance.Validate(); err != nil {
		return fmt.Errorf("invalid account balance: %w", err)
	}

	query := `
		INSERT INTO account_balances (timestamp, currency, balance, available, frozen, equity)
		VALUES (?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(query,
		balance.Timestamp.UTC(),
		balance.Currency,
		balance.Balance,
		balance.Available,
		balance.Frozen,
		balance.Equity,
	)
	if err != nil {
		return fmt.Errorf("failed to insert account balance: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	balance.ID = id
	return nil
}

// InsertPosition 插入持仓记录 / Insert position record
// 将持仓数据写入positions表，记录合约、持仓量、盈亏等信息
// Write position data to positions table with contract, position size, PnL info
//
// Parameters:
//   - position: Position data model, must contain valid Instrument, PositionSide, PositionSize fields
//     Timestamp will be converted to UTC for storage
//
// Returns:
//   - error: 数据验证失败或数据库写入失败时返回错误 / Error on validation failure or database write failure
//     成功时会将生成的ID回写到position.ID字段 / On success, generated ID is written back to position.ID
func (s *Storage) InsertPosition(position *models.Position) error {
	if err := position.Validate(); err != nil {
		return fmt.Errorf("invalid position: %w", err)
	}

	query := `
		INSERT INTO positions (timestamp, instrument, position_side, position_size, average_price, unrealized_pnl, margin, leverage)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.Exec(query,
		position.Timestamp.UTC(),
		position.Instrument,
		position.PositionSide,
		position.PositionSize,
		position.AveragePrice,
		position.UnrealizedPnL,
		position.Margin,
		position.Leverage,
	)
	if err != nil {
		return fmt.Errorf("failed to insert position: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	position.ID = id
	return nil
}

// GetLatestAccountBalances 获取最新的账户余额 / Get latest account balances
// 查询最新时间戳的所有币种账户余额记录
// Query all currency account balance records with the latest timestamp
//
// Returns:
//   - []models.AccountBalance: 最新的账户余额切片，按币种排序
//     Slice of latest account balances, sorted by currency
//     如果没有记录，返回空切片 / Returns empty slice if no records
//   - error: 数据库查询失败时返回错误 / Error on database query failure
func (s *Storage) GetLatestAccountBalances() ([]models.AccountBalance, error) {
	query := `
		SELECT id, timestamp, currency, balance, available, frozen, equity
		FROM account_balances
		WHERE timestamp = (SELECT MAX(timestamp) FROM account_balances)
		ORDER BY currency
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest balances: %w", err)
	}
	defer rows.Close()

	var balances []models.AccountBalance
	for rows.Next() {
		var b models.AccountBalance
		var timestamp string
		if err := rows.Scan(&b.ID, &timestamp, &b.Currency, &b.Balance, &b.Available, &b.Frozen, &b.Equity); err != nil {
			return nil, fmt.Errorf("failed to scan balance: %w", err)
		}

		// Parse timestamp (SQLite stores in RFC3339 format)
		b.Timestamp, err = time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
		}

		balances = append(balances, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return balances, nil
}

// GetLatestPositions 获取最新的持仓 / Get latest positions
func (s *Storage) GetLatestPositions() ([]models.Position, error) {
	query := `
		SELECT id, timestamp, instrument, position_side, position_size, average_price, unrealized_pnl, margin, leverage
		FROM positions
		WHERE timestamp = (SELECT MAX(timestamp) FROM positions)
		ORDER BY instrument
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest positions: %w", err)
	}
	defer rows.Close()

	var positions []models.Position
	for rows.Next() {
		var p models.Position
		var timestamp string
		if err := rows.Scan(&p.ID, &timestamp, &p.Instrument, &p.PositionSide, &p.PositionSize, &p.AveragePrice, &p.UnrealizedPnL, &p.Margin, &p.Leverage); err != nil {
			return nil, fmt.Errorf("failed to scan position: %w", err)
		}

		// Parse timestamp (SQLite stores in RFC3339 format)
		p.Timestamp, err = time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
		}

		positions = append(positions, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return positions, nil
}

// GetAccountBalancesByTimeRange 按时间范围查询账户余额 / Query account balances by time range
func (s *Storage) GetAccountBalancesByTimeRange(currency string, startTime, endTime time.Time) ([]models.AccountBalance, error) {
	query := `
		SELECT id, timestamp, currency, balance, available, frozen, equity
		FROM account_balances
		WHERE currency = ? AND timestamp BETWEEN ? AND ?
		ORDER BY timestamp ASC
	`

	rows, err := s.db.Query(query, currency, startTime.UTC(), endTime.UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to query balances by time range: %w", err)
	}
	defer rows.Close()

	var balances []models.AccountBalance
	for rows.Next() {
		var b models.AccountBalance
		var timestamp string
		if err := rows.Scan(&b.ID, &timestamp, &b.Currency, &b.Balance, &b.Available, &b.Frozen, &b.Equity); err != nil {
			return nil, fmt.Errorf("failed to scan balance: %w", err)
		}

		// Parse timestamp (SQLite stores in RFC3339 format)
		b.Timestamp, err = time.Parse(time.RFC3339, timestamp)
		if err != nil {
			return nil, fmt.Errorf("failed to parse timestamp: %w", err)
		}

		balances = append(balances, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return balances, nil
}

// Close 关闭数据库连接 / Close database connection
func (s *Storage) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// HealthCheck 健康检查 / Health check for database connectivity
func (s *Storage) HealthCheck() error {
	return s.db.Ping()
}
