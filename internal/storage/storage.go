package storage

import (
	"context"
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
		leverage REAL,
		margin_mode VARCHAR(10) DEFAULT 'cross'
	);
	CREATE INDEX IF NOT EXISTS idx_positions_timestamp ON positions(timestamp);
	CREATE INDEX IF NOT EXISTS idx_positions_timestamp_instrument ON positions(timestamp, instrument);
	`

	if _, err := s.db.Exec(positionsSchema); err != nil {
		return fmt.Errorf("failed to create positions table: %w", err)
	}

	// Create order_history table (Feature 3: Order Control)
	orderHistorySchema := `
	CREATE TABLE IF NOT EXISTS order_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		order_id TEXT NOT NULL,
		inst_id TEXT NOT NULL,
		side TEXT NOT NULL,
		ord_type TEXT NOT NULL,
		size TEXT NOT NULL,
		price TEXT,
		reduce_only BOOLEAN NOT NULL DEFAULT 0,
		placed_at DATETIME NOT NULL,
		week_start DATE NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_order_history_week_start ON order_history(week_start);
	CREATE INDEX IF NOT EXISTS idx_order_history_order_id ON order_history(order_id);
	CREATE INDEX IF NOT EXISTS idx_order_history_placed_at ON order_history(placed_at);
	`

	if _, err := s.db.Exec(orderHistorySchema); err != nil {
		return fmt.Errorf("failed to create order_history table: %w", err)
	}

	// Create positions_history table (V3.2: Historical Positions)
	positionsHistorySchema := `
	CREATE TABLE IF NOT EXISTS positions_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		pos_id TEXT NOT NULL UNIQUE,
		inst_type TEXT NOT NULL,
		inst_id TEXT NOT NULL,
		mgn_mode TEXT NOT NULL,
		pos_side TEXT NOT NULL,
		lever TEXT NOT NULL,
		open_avg_px TEXT NOT NULL,
		close_avg_px TEXT NOT NULL,
		open_max_pos TEXT NOT NULL,
		close_total_pos TEXT NOT NULL,
		realized_pnl TEXT NOT NULL,
		pnl TEXT NOT NULL,
		pnl_ratio TEXT NOT NULL,
		fee TEXT NOT NULL,
		funding_fee TEXT NOT NULL,
		liq_penalty TEXT NOT NULL,
		close_type TEXT NOT NULL,
		direction TEXT NOT NULL,
		ccy TEXT NOT NULL,
		opened_at DATETIME NOT NULL,
		closed_at DATETIME NOT NULL,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);
	CREATE INDEX IF NOT EXISTS idx_positions_history_pos_id ON positions_history(pos_id);
	CREATE INDEX IF NOT EXISTS idx_positions_history_inst_id ON positions_history(inst_id);
	CREATE INDEX IF NOT EXISTS idx_positions_history_closed_at ON positions_history(closed_at);
	CREATE INDEX IF NOT EXISTS idx_positions_history_close_type ON positions_history(close_type);
	`

	if _, err := s.db.Exec(positionsHistorySchema); err != nil {
		return fmt.Errorf("failed to create positions_history table: %w", err)
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
		INSERT INTO positions (timestamp, instrument, position_side, position_size, average_price, unrealized_pnl, margin, leverage, margin_mode)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		position.MarginMode,
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
// Returns only positions from the most recent snapshot.
// NOTE: This function is now mainly used for historical data analysis.
// TPSL service fetches positions directly from OKX API for real-time accuracy.
func (s *Storage) GetLatestPositions() ([]models.Position, error) {
	// First, get the latest timestamp
	var latestTimestamp string
	err := s.db.QueryRow("SELECT MAX(timestamp) FROM positions").Scan(&latestTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest timestamp: %w", err)
	}

	// If no positions in database yet, return empty slice
	if latestTimestamp == "" {
		return []models.Position{}, nil
	}

	query := `
		SELECT id, timestamp, instrument, position_side, position_size, average_price, unrealized_pnl, margin, leverage, margin_mode
		FROM positions
		WHERE timestamp = ?
		ORDER BY instrument
	`

	rows, err := s.db.Query(query, latestTimestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest positions: %w", err)
	}
	defer rows.Close()

	var positions []models.Position
	for rows.Next() {
		var p models.Position
		var timestamp string
		if err := rows.Scan(&p.ID, &timestamp, &p.Instrument, &p.PositionSide, &p.PositionSize, &p.AveragePrice, &p.UnrealizedPnL, &p.Margin, &p.Leverage, &p.MarginMode); err != nil {
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

// ============================================================================
// Order History Methods (Phase 1B)
// ============================================================================
// TODO: Implement actual database operations for order history tracking
// ============================================================================

// InsertOrderHistory inserts an order history record into storage
// 插入订单历史记录到数据库 / Insert order history record to database
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - order: Order history data model, must contain valid OrderID, InstId, Side, OrdType, Size fields
//
// Returns:
//   - error: 数据验证失败或数据库写入失败时返回错误 / Error on validation failure or database write failure
func (s *Storage) InsertOrderHistory(ctx context.Context, order *models.OrderHistory) error {
	if err := order.Validate(); err != nil {
		return fmt.Errorf("invalid order history: %w", err)
	}

	// Check if order with this order_id already exists to avoid duplicates
	// This is important when syncing orders from OKX API
	var existingID int64
	checkQuery := `SELECT id FROM order_history WHERE order_id = ? LIMIT 1`
	err := s.db.QueryRowContext(ctx, checkQuery, order.OrderID).Scan(&existingID)

	if err == nil {
		// Order already exists, skip insertion
		order.ID = existingID
		return nil // Not an error - idempotent operation
	} else if err != sql.ErrNoRows {
		// Real error occurred during check
		return fmt.Errorf("failed to check existing order: %w", err)
	}

	// Order doesn't exist, proceed with insertion
	query := `
		INSERT INTO order_history (order_id, inst_id, side, ord_type, size, price, reduce_only, placed_at, week_start, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.ExecContext(ctx, query,
		order.OrderID,
		order.InstId,
		order.Side,
		order.OrdType,
		order.Size,
		order.Price,
		order.ReduceOnly,
		order.PlacedAt.UTC(),
		order.WeekStart.UTC(),
		order.Status,
		order.CreatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert order history: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	order.ID = id
	return nil
}

// GetOrderCountForWeek returns the count of orders placed in a given week
// 获取指定周的订单数量 / Get order count for specified week
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - weekStart: Start of the week (Monday 00:00:00 UTC)
//
// Returns:
//   - int: 该周的订单数量 / Order count for the week
//   - error: 数据库查询失败时返回错误 / Error on database query failure
func (s *Storage) GetOrderCountForWeek(ctx context.Context, weekStart time.Time) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM order_history
		WHERE week_start = ?
	`

	var count int
	err := s.db.QueryRowContext(ctx, query, weekStart.UTC()).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get order count for week: %w", err)
	}

	return count, nil
}

// GetOrdersForWeek retrieves all orders placed in a given week
// 获取指定周的所有订单 / Get all orders for specified week
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - weekStart: Start of the week (Monday 00:00:00 UTC)
//
// Returns:
//   - []models.OrderHistory: 该周的订单列表，按下单时间排序 / Order list for the week, sorted by placed_at
//   - error: 数据库查询失败时返回错误 / Error on database query failure
func (s *Storage) GetOrdersForWeek(ctx context.Context, weekStart time.Time) ([]models.OrderHistory, error) {
	query := `
		SELECT id, order_id, inst_id, side, ord_type, size, price, reduce_only, placed_at, week_start, status, created_at
		FROM order_history
		WHERE week_start = ?
		ORDER BY placed_at ASC
	`

	rows, err := s.db.QueryContext(ctx, query, weekStart.UTC())
	if err != nil {
		return nil, fmt.Errorf("failed to query orders for week: %w", err)
	}
	defer rows.Close()

	var orders []models.OrderHistory
	for rows.Next() {
		var order models.OrderHistory
		var placedAt, weekStartStr, createdAt string
		var price sql.NullString

		err := rows.Scan(
			&order.ID,
			&order.OrderID,
			&order.InstId,
			&order.Side,
			&order.OrdType,
			&order.Size,
			&price,
			&order.ReduceOnly,
			&placedAt,
			&weekStartStr,
			&order.Status,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan order: %w", err)
		}

		// Parse timestamps
		order.PlacedAt, err = time.Parse(time.RFC3339, placedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse placed_at: %w", err)
		}

		order.WeekStart, err = time.Parse(time.RFC3339, weekStartStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse week_start: %w", err)
		}

		order.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}

		// Handle nullable price
		if price.Valid {
			order.Price = price.String
		}

		orders = append(orders, order)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return orders, nil
}

// GetWeeklyOrderStats retrieves aggregated order statistics for a week
// 获取指定周的订单统计信息 / Get aggregated order statistics for specified week
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - weekStart: Start of the week (Monday 00:00:00 UTC)
//
// Returns:
//   - *models.WeeklyOrderCount: 周订单统计数据 / Weekly order statistics
//   - error: 数据库查询失败时返回错误 / Error on database query failure
func (s *Storage) GetWeeklyOrderStats(ctx context.Context, weekStart time.Time) (*models.WeeklyOrderCount, error) {
	query := `
		SELECT
			COUNT(*) as total_orders,
			COALESCE(SUM(CASE WHEN reduce_only = 1 THEN 1 ELSE 0 END), 0) as reduce_only_orders,
			COALESCE(SUM(CASE WHEN status = 'placed' THEN 1 ELSE 0 END), 0) as placed_orders,
			COALESCE(SUM(CASE WHEN status = 'filled' THEN 1 ELSE 0 END), 0) as filled_orders,
			COALESCE(SUM(CASE WHEN status = 'canceled' THEN 1 ELSE 0 END), 0) as canceled_orders,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed_orders
		FROM order_history
		WHERE week_start = ?
	`

	stats := &models.WeeklyOrderCount{
		WeekStart: weekStart.UTC(),
	}

	err := s.db.QueryRowContext(ctx, query, weekStart.UTC()).Scan(
		&stats.TotalOrders,
		&stats.ReduceOnlyOrders,
		&stats.PlacedOrders,
		&stats.FilledOrders,
		&stats.CanceledOrders,
		&stats.FailedOrders,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get weekly order stats: %w", err)
	}

	// Calculate RegularOrders (Total - ReduceOnly)
	stats.RegularOrders = stats.TotalOrders - stats.ReduceOnlyOrders

	return stats, nil
}

// ============================================================================
// Pending Confirmation Methods (Phase 1B)
// ============================================================================
// TODO: Implement actual database operations for pending confirmations
// ============================================================================

// InsertPendingConfirmation inserts a pending confirmation record
func (s *Storage) InsertPendingConfirmation(ctx context.Context, conf *models.PendingConfirmation) error {
	// TODO: Implement database INSERT
	return fmt.Errorf("InsertPendingConfirmation not yet implemented")
}

// GetPendingConfirmationsDue retrieves confirmations that need checking
func (s *Storage) GetPendingConfirmationsDue(ctx context.Context, now time.Time) ([]models.PendingConfirmation, error) {
	// TODO: Implement database query with time filter
	return nil, fmt.Errorf("GetPendingConfirmationsDue not yet implemented")
}

// GetPendingConfirmation retrieves a specific pending confirmation by order ID
func (s *Storage) GetPendingConfirmation(ctx context.Context, orderID string) (*models.PendingConfirmation, error) {
	// TODO: Implement database query by orderID
	return nil, fmt.Errorf("GetPendingConfirmation not yet implemented")
}

// UpdatePendingConfirmation updates a pending confirmation record
func (s *Storage) UpdatePendingConfirmation(ctx context.Context, orderID string, update *models.ConfirmationUpdate) error {
	// TODO: Implement database UPDATE
	return fmt.Errorf("UpdatePendingConfirmation not yet implemented")
}

// DeletePendingConfirmation removes a pending confirmation record
func (s *Storage) DeletePendingConfirmation(ctx context.Context, orderID string) error {
	// TODO: Implement database DELETE
	return fmt.Errorf("DeletePendingConfirmation not yet implemented")
}

// ============================================================================
// Position History Methods (V3.2)
// ============================================================================

// InsertPositionHistory inserts a position history record into storage
// 插入历史仓位记录到数据库 / Insert position history record to database
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - position: Position history data model
//
// Returns:
//   - error: 数据验证失败或数据库写入失败时返回错误 / Error on validation failure or database write failure
func (s *Storage) InsertPositionHistory(ctx context.Context, position *models.PositionHistory) error {
	if err := position.Validate(); err != nil {
		return fmt.Errorf("invalid position history: %w", err)
	}

	// Check if position with this pos_id already exists to avoid duplicates
	var existingID int64
	checkQuery := `SELECT id FROM positions_history WHERE pos_id = ? LIMIT 1`
	err := s.db.QueryRowContext(ctx, checkQuery, position.PosId).Scan(&existingID)

	if err == nil {
		// Position already exists, skip insertion
		position.ID = existingID
		return nil // Not an error - idempotent operation
	} else if err != sql.ErrNoRows {
		// Real error occurred during check
		return fmt.Errorf("failed to check existing position: %w", err)
	}

	// Position doesn't exist, proceed with insertion
	query := `
		INSERT INTO positions_history (
			pos_id, inst_type, inst_id, mgn_mode, pos_side, lever,
			open_avg_px, close_avg_px, open_max_pos, close_total_pos,
			realized_pnl, pnl, pnl_ratio, fee, funding_fee, liq_penalty,
			close_type, direction, ccy, opened_at, closed_at, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	result, err := s.db.ExecContext(ctx, query,
		position.PosId,
		position.InstType,
		position.InstId,
		position.MgnMode,
		position.PosSide,
		position.Lever,
		position.OpenAvgPx,
		position.CloseAvgPx,
		position.OpenMaxPos,
		position.CloseTotalPos,
		position.RealizedPnl,
		position.Pnl,
		position.PnlRatio,
		position.Fee,
		position.FundingFee,
		position.LiqPenalty,
		position.CloseType,
		position.Direction,
		position.Ccy,
		position.OpenedAt.UTC(),
		position.ClosedAt.UTC(),
		position.CreatedAt.UTC(),
	)
	if err != nil {
		return fmt.Errorf("failed to insert position history: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	position.ID = id
	return nil
}

// GetPositionsHistory retrieves position history with optional filters
// 获取历史仓位列表，支持按币种和时间过滤 / Get position history list with optional currency and time filters
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - instId: Optional instrument ID filter (empty string = all instruments)
//   - limit: Maximum number of records to return (0 = no limit)
//
// Returns:
//   - []models.PositionHistory: 历史仓位列表，按关闭时间倒序 / Position history list, sorted by closed_at DESC
//   - error: 数据库查询失败时返回错误 / Error on database query failure
func (s *Storage) GetPositionsHistory(ctx context.Context, instId string, limit int) ([]models.PositionHistory, error) {
	query := `
		SELECT id, pos_id, inst_type, inst_id, mgn_mode, pos_side, lever,
			   open_avg_px, close_avg_px, open_max_pos, close_total_pos,
			   realized_pnl, pnl, pnl_ratio, fee, funding_fee, liq_penalty,
			   close_type, direction, ccy, opened_at, closed_at, created_at
		FROM positions_history
	`

	var args []any
	if instId != "" {
		query += ` WHERE inst_id = ?`
		args = append(args, instId)
	}

	query += ` ORDER BY closed_at DESC`

	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query positions history: %w", err)
	}
	defer rows.Close()

	var positions []models.PositionHistory
	for rows.Next() {
		var position models.PositionHistory
		var openedAt, closedAt, createdAt string

		err := rows.Scan(
			&position.ID,
			&position.PosId,
			&position.InstType,
			&position.InstId,
			&position.MgnMode,
			&position.PosSide,
			&position.Lever,
			&position.OpenAvgPx,
			&position.CloseAvgPx,
			&position.OpenMaxPos,
			&position.CloseTotalPos,
			&position.RealizedPnl,
			&position.Pnl,
			&position.PnlRatio,
			&position.Fee,
			&position.FundingFee,
			&position.LiqPenalty,
			&position.CloseType,
			&position.Direction,
			&position.Ccy,
			&openedAt,
			&closedAt,
			&createdAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan position: %w", err)
		}

		// Parse timestamps
		position.OpenedAt, err = time.Parse(time.RFC3339, openedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse opened_at: %w", err)
		}

		position.ClosedAt, err = time.Parse(time.RFC3339, closedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse closed_at: %w", err)
		}

		position.CreatedAt, err = time.Parse(time.RFC3339, createdAt)
		if err != nil {
			return nil, fmt.Errorf("failed to parse created_at: %w", err)
		}

		positions = append(positions, position)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return positions, nil
}
