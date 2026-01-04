package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

// Storage GORM存储实现 / GORM storage implementation
type Storage struct {
	db *gorm.DB
}

// New 创建GORM存储实例 / Create GORM storage instance
// 初始化GORM数据库连接，配置连接池，创建表结构
// Initialize GORM database connection, configure connection pool, create table schema
//
// Parameters:
//   - dbPath: Database file path (e.g., "./data/tenyojubaku.db")
//   - walMode: Enable WAL mode for better concurrency
//   - maxOpenConns: Maximum open connections
//   - maxIdleConns: Maximum idle connections
//
// Returns:
//   - *Storage: 已初始化的GORM存储实例 / Initialized GORM storage instance
//   - error: 初始化失败时返回错误 / Error on initialization failure
func New(dbPath string, walMode bool, maxOpenConns, maxIdleConns int) (*Storage, error) {
	// Ensure database directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %w", err)
	}

	// Open GORM connection
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		SkipDefaultTransaction: true,  // 跳过默认事务，提升性能 / Skip default transaction for better performance
		PrepareStmt:            true,  // 缓存prepared statements / Cache prepared statements
		Logger:                 logger.Default.LogMode(logger.Silent), // 生产环境静默日志 / Silent logs in production
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open GORM database: %w", err)
	}

	// Get underlying *sql.DB for connection pool settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying database: %w", err)
	}

	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetMaxIdleConns(maxIdleConns)

	// Enable WAL mode if requested
	if walMode {
		db.Exec("PRAGMA journal_mode=WAL")
	}

	// Performance optimizations for SQLite
	db.Exec("PRAGMA synchronous=NORMAL")        // 平衡性能和安全性 / Balance performance and safety
	db.Exec("PRAGMA cache_size=-64000")         // 64MB缓存 / 64MB cache
	db.Exec("PRAGMA temp_store=MEMORY")         // 临时表存内存 / Temp tables in memory
	db.Exec("PRAGMA mmap_size=268435456")       // 256MB内存映射 / 256MB memory-mapped I/O

	storage := &Storage{db: db}

	// Initialize schema
	if err := storage.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return storage, nil
}

// initSchema 初始化数据库架构 / Initialize database schema
func (s *Storage) initSchema() error {
	// AutoMigrate creates tables from struct definitions
	err := s.db.AutoMigrate(
		&models.AccountBalance{},
		&models.Position{},
		&models.OrderHistory{},
		&models.PositionHistory{},
		// NOTE: pending_confirmations表不在此迁移中创建
		// 将在Feature 3 Phase 1B实现时添加
	)
	if err != nil {
		return fmt.Errorf("failed to auto-migrate schema: %w", err)
	}

	// Create unique indexes for idempotent inserts
	migrator := s.db.Migrator()

	// Order history unique index (for idempotent inserts)
	if !migrator.HasIndex(&models.OrderHistory{}, "idx_order_history_order_id_unique") {
		s.db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_order_history_order_id_unique ON order_history(order_id)")
	}

	// Position history unique index (for idempotent inserts)
	if !migrator.HasIndex(&models.PositionHistory{}, "idx_positions_history_pos_id") {
		s.db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_positions_history_pos_id ON positions_history(pos_id)")
	}

	// Composite index for positions (timestamp + instrument)
	if !migrator.HasIndex(&models.Position{}, "idx_positions_timestamp_instrument") {
		s.db.Exec("CREATE INDEX IF NOT EXISTS idx_positions_timestamp_instrument ON positions(timestamp, instrument)")
	}

	return nil
}

// ============================================================================
// Account Balance Operations
// ============================================================================

// InsertAccountBalance 插入账户余额记录 / Insert account balance record
// 将账户余额数据写入account_balances表，记录时间戳和币种余额信息
// Write account balance data to account_balances table with timestamp and currency balance info
//
// Parameters:
//   - balance: Account balance data model, must contain valid Currency, Balance, Available fields
//
// Returns:
//   - error: 数据验证失败或数据库写入失败时返回错误 / Error on validation failure or database write failure
func (s *Storage) InsertAccountBalance(balance *models.AccountBalance) error {
	if err := balance.Validate(); err != nil {
		return fmt.Errorf("invalid account balance: %w", err)
	}

	result := s.db.Create(balance)
	return result.Error
}

// GetLatestAccountBalances 获取最新的账户余额 / Get latest account balances
// 查询最新时间戳的所有币种账户余额记录
// Query all currency account balance records with the latest timestamp
//
// Returns:
//   - []models.AccountBalance: 最新的账户余额切片，按币种排序 / Slice of latest account balances, sorted by currency
//   - error: 数据库查询失败时返回错误 / Error on database query failure
func (s *Storage) GetLatestAccountBalances() ([]models.AccountBalance, error) {
	var balances []models.AccountBalance

	// Subquery to get latest timestamp
	subQuery := s.db.Model(&models.AccountBalance{}).
		Select("MAX(timestamp)")

	// Main query: filter by latest timestamp
	err := s.db.Where("timestamp = (?)", subQuery).
		Order("currency").
		Find(&balances).Error

	return balances, err
}

// GetAccountBalancesByTimeRange 按时间范围查询账户余额 / Query account balances by time range
// 查询指定币种在时间范围内的账户余额记录
// Query account balance records for specified currency within time range
//
// Parameters:
//   - currency: Currency to query (e.g., "BTC", "USDT")
//   - startTime: Start time of the range (inclusive)
//   - endTime: End time of the range (inclusive)
//
// Returns:
//   - []models.AccountBalance: 账户余额记录切片，按时间升序 / Account balance records, ordered by time ASC
//   - error: 数据库查询失败时返回错误 / Error on database query failure
func (s *Storage) GetAccountBalancesByTimeRange(currency string, startTime, endTime time.Time) ([]models.AccountBalance, error) {
	var balances []models.AccountBalance

	err := s.db.Where("currency = ? AND timestamp BETWEEN ? AND ?", currency, startTime.UTC(), endTime.UTC()).
		Order("timestamp ASC").
		Find(&balances).Error

	return balances, err
}

// ============================================================================
// Position Operations
// ============================================================================

// InsertPosition 插入持仓记录 / Insert position record
// 将持仓数据写入positions表，记录合约、持仓量、盈亏等信息
// Write position data to positions table with contract, position size, PnL info
//
// Parameters:
//   - position: Position data model, must contain valid Instrument, PositionSide, PositionSize fields
//
// Returns:
//   - error: 数据验证失败或数据库写入失败时返回错误 / Error on validation failure or database write failure
func (s *Storage) InsertPosition(position *models.Position) error {
	if err := position.Validate(); err != nil {
		return fmt.Errorf("invalid position: %w", err)
	}

	result := s.db.Create(position)
	return result.Error
}

// GetLatestPositions 获取最新的持仓 / Get latest positions
// 查询最新时间戳的所有持仓记录
// Query all position records with the latest timestamp
//
// NOTE: This function is mainly used for historical data analysis.
// TPSL service fetches positions directly from OKX API for real-time accuracy.
//
// Returns:
//   - []models.Position: 最新的持仓切片，按合约排序 / Slice of latest positions, sorted by instrument
//   - error: 数据库查询失败时返回错误 / Error on database query failure
func (s *Storage) GetLatestPositions() ([]models.Position, error) {
	// Use subquery to get positions with the latest timestamp
	var positions []models.Position

	subQuery := s.db.Model(&models.Position{}).
		Select("MAX(timestamp)")

	err := s.db.Where("timestamp = (?)", subQuery).
		Order("instrument").
		Find(&positions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to query latest positions: %w", err)
	}

	return positions, nil
}

// ============================================================================
// Order History Operations
// ============================================================================

// InsertOrderHistory 插入订单历史记录 / Insert order history record
// 将订单历史数据写入order_history表，使用幂等插入（基于order_id）
// Write order history data to order_history table with idempotent insert (based on order_id)
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - order: Order history data model, must contain valid OrderID, InstId, Side, Size fields
//
// Returns:
//   - error: 数据验证失败或数据库写入失败时返回错误 / Error on validation failure or database write failure
//     成功时会将生成的ID回写到order.ID字段 / On success, generated ID is written back to order.ID
//     幂等：如果order_id已存在，返回现有记录的ID / Idempotent: if order_id exists, returns existing record ID
func (s *Storage) InsertOrderHistory(ctx context.Context, order *models.OrderHistory) error {
	if err := order.Validate(); err != nil {
		return fmt.Errorf("invalid order history: %w", err)
	}

	// FirstOrCreate: find existing record OR create new one
	// This provides idempotent insert behavior
	result := s.db.WithContext(ctx).
		Where("order_id = ?", order.OrderID).
		FirstOrCreate(order)

	return result.Error
}

// GetOrderCountForWeek 获取指定周的订单数量 / Get order count for specified week
// 统计指定周的订单总数
// Count total orders for specified week
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - weekStart: Start of the week (Monday 00:00:00 UTC)
//
// Returns:
//   - int: 该周的订单数量 / Order count for the week
//   - error: 数据库查询失败时返回错误 / Error on database query failure
func (s *Storage) GetOrderCountForWeek(ctx context.Context, weekStart time.Time) (int, error) {
	var count int64

	err := s.db.WithContext(ctx).
		Model(&models.OrderHistory{}).
		Where("week_start = ?", weekStart.UTC()).
		Count(&count).Error

	return int(count), err
}

// GetOrdersForWeek 获取指定周的所有订单 / Get all orders for specified week
// 查询指定周的订单列表，按下单时间排序
// Query order list for specified week, sorted by placed_at
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - weekStart: Start of the week (Monday 00:00:00 UTC)
//
// Returns:
//   - []models.OrderHistory: 该周的订单列表 / Order list for the week
//   - error: 数据库查询失败时返回错误 / Error on database query failure
func (s *Storage) GetOrdersForWeek(ctx context.Context, weekStart time.Time) ([]models.OrderHistory, error) {
	var orders []models.OrderHistory

	err := s.db.WithContext(ctx).
		Where("week_start = ?", weekStart.UTC()).
		Order("placed_at ASC").
		Find(&orders).Error

	return orders, err
}

// GetWeeklyOrderStats 获取指定周的订单统计信息 / Get aggregated order statistics for specified week
// 查询指定周的订单统计数据，包括按状态分类的订单数量
// Query aggregated order statistics for specified week, including counts by status
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - weekStart: Start of the week (Monday 00:00:00 UTC)
//
// Returns:
//   - *models.WeeklyOrderCount: 周订单统计数据 / Weekly order statistics
//   - error: 数据库查询失败时返回错误 / Error on database query failure
func (s *Storage) GetWeeklyOrderStats(ctx context.Context, weekStart time.Time) (*models.WeeklyOrderCount, error) {
	stats := &models.WeeklyOrderCount{
		WeekStart: weekStart.UTC(),
	}

	// Temporary struct for aggregation result
	type AggResult struct {
		TotalOrders      int
		ReduceOnlyOrders int
		PlacedOrders     int
		FilledOrders     int
		CanceledOrders   int
		FailedOrders     int
	}

	var result AggResult
	err := s.db.Model(&models.OrderHistory{}).
		WithContext(ctx).
		Select(`
			COUNT(*) as total_orders,
			COALESCE(SUM(CASE WHEN reduce_only = 1 THEN 1 ELSE 0 END), 0) as reduce_only_orders,
			COALESCE(SUM(CASE WHEN status = 'placed' THEN 1 ELSE 0 END), 0) as placed_orders,
			COALESCE(SUM(CASE WHEN status = 'filled' THEN 1 ELSE 0 END), 0) as filled_orders,
			COALESCE(SUM(CASE WHEN status = 'canceled' THEN 1 ELSE 0 END), 0) as canceled_orders,
			COALESCE(SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END), 0) as failed_orders
		`).
		Where("week_start = ?", weekStart.UTC()).
		Scan(&result).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get weekly order stats: %w", err)
	}

	// Map result to stats
	stats.TotalOrders = result.TotalOrders
	stats.ReduceOnlyOrders = result.ReduceOnlyOrders
	stats.PlacedOrders = result.PlacedOrders
	stats.FilledOrders = result.FilledOrders
	stats.CanceledOrders = result.CanceledOrders
	stats.FailedOrders = result.FailedOrders
	stats.RegularOrders = stats.TotalOrders - stats.ReduceOnlyOrders

	return stats, nil
}

// ============================================================================
// Position History Operations
// ============================================================================

// InsertPositionHistory 插入历史仓位记录 / Insert position history record
// 将历史仓位数据写入positions_history表，使用幂等插入（基于pos_id）
// Write position history data to positions_history table with idempotent insert (based on pos_id)
//
// Parameters:
//   - ctx: Context for cancellation and timeout
//   - position: Position history data model
//
// Returns:
//   - error: 数据验证失败或数据库写入失败时返回错误 / Error on validation failure or database write failure
//     幂等：如果pos_id已存在，返回现有记录的ID / Idempotent: if pos_id exists, returns existing record ID
func (s *Storage) InsertPositionHistory(ctx context.Context, position *models.PositionHistory) error {
	if err := position.Validate(); err != nil {
		return fmt.Errorf("invalid position history: %w", err)
	}

	// FirstOrCreate: find existing record OR create new one
	// This provides idempotent insert behavior
	result := s.db.WithContext(ctx).
		Where("pos_id = ?", position.PosId).
		FirstOrCreate(position)

	return result.Error
}

// GetPositionsHistory 获取历史仓位列表 / Get position history list
// 查询历史仓位记录，支持按币种和数量限制过滤
// Query position history records with optional filters for instrument and limit
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
	var positions []models.PositionHistory

	query := s.db.WithContext(ctx)

	// Optional filter: instrument ID
	if instId != "" {
		query = query.Where("inst_id = ?", instId)
	}

	// Order by closed_at DESC
	query = query.Order("closed_at DESC")

	// Optional limit
	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&positions).Error
	return positions, err
}

// ============================================================================
// Pending Confirmation Operations (NOT IMPLEMENTED)
// ============================================================================
// These 5 methods are not implemented in this migration.
// They will be implemented in Feature 3 Phase 1B.
// ============================================================================

// InsertPendingConfirmation 插入待确认订单记录 / Insert pending confirmation record
// NOTE: Not yet implemented - will be added in Feature 3 Phase 1B
func (s *Storage) InsertPendingConfirmation(ctx context.Context, conf *models.PendingConfirmation) error {
	return fmt.Errorf("InsertPendingConfirmation not yet implemented")
}

// GetPendingConfirmationsDue 获取到期的待确认订单 / Get pending confirmations due
// NOTE: Not yet implemented - will be added in Feature 3 Phase 1B
func (s *Storage) GetPendingConfirmationsDue(ctx context.Context, now time.Time) ([]models.PendingConfirmation, error) {
	return nil, fmt.Errorf("GetPendingConfirmationsDue not yet implemented")
}

// GetPendingConfirmation 获取指定订单的待确认记录 / Get pending confirmation by order ID
// NOTE: Not yet implemented - will be added in Feature 3 Phase 1B
func (s *Storage) GetPendingConfirmation(ctx context.Context, orderID string) (*models.PendingConfirmation, error) {
	return nil, fmt.Errorf("GetPendingConfirmation not yet implemented")
}

// UpdatePendingConfirmation 更新待确认订单记录 / Update pending confirmation record
// NOTE: Not yet implemented - will be added in Feature 3 Phase 1B
func (s *Storage) UpdatePendingConfirmation(ctx context.Context, orderID string, update *models.ConfirmationUpdate) error {
	return fmt.Errorf("UpdatePendingConfirmation not yet implemented")
}

// DeletePendingConfirmation 删除待确认订单记录 / Delete pending confirmation record
// NOTE: Not yet implemented - will be added in Feature 3 Phase 1B
func (s *Storage) DeletePendingConfirmation(ctx context.Context, orderID string) error {
	return fmt.Errorf("DeletePendingConfirmation not yet implemented")
}

// ============================================================================
// Utility Operations
// ============================================================================

// Close 关闭数据库连接 / Close database connection
// 优雅关闭数据库连接，释放资源
// Gracefully close database connection and release resources
//
// Returns:
//   - error: 关闭失败时返回错误 / Error on closure failure
func (s *Storage) Close() error {
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err != nil {
			return fmt.Errorf("failed to get underlying database: %w", err)
		}
		return sqlDB.Close()
	}
	return nil
}

// HealthCheck 健康检查 / Health check for database connectivity
// 检查数据库连接是否正常
// Check if database connection is healthy
//
// Returns:
//   - error: 连接失败时返回错误 / Error on connection failure
func (s *Storage) HealthCheck() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database: %w", err)
	}
	return sqlDB.Ping()
}
