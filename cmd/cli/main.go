package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"github.com/wTHU1Ew/TenyoJubaku/internal/config"
	"github.com/wTHU1Ew/TenyoJubaku/internal/logger"
	"github.com/wTHU1Ew/TenyoJubaku/internal/notifier"
	"github.com/wTHU1Ew/TenyoJubaku/internal/okx"
	"github.com/wTHU1Ew/TenyoJubaku/internal/ordercontrol"
	"github.com/wTHU1Ew/TenyoJubaku/internal/storage"
	"github.com/wTHU1Ew/TenyoJubaku/pkg/models"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "order":
		handleOrderCommand()
	case "position":
		handlePositionCommand()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func handleOrderCommand() {
	if len(os.Args) < 3 {
		printOrderUsage()
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "place":
		handleOrderPlace()
	case "stats":
		handleOrderStats()
	case "list":
		handleOrderList()
	default:
		fmt.Fprintf(os.Stderr, "Unknown order subcommand: %s\n\n", subcommand)
		printOrderUsage()
		os.Exit(1)
	}
}

func handleOrderPlace() {
	fs := flag.NewFlagSet("order place", flag.ExitOnError)

	instrument := fs.String("instrument", "", "Instrument ID (e.g., BTC-USDT-SWAP)")
	side := fs.String("side", "", "Order side: buy or sell")
	orderType := fs.String("type", "limit", "Order type: limit, post_only, market")
	size := fs.String("size", "", "Order size")
	price := fs.String("price", "", "Order price (required for limit orders)")
	reduceOnly := fs.Bool("reduce-only", false, "Reduce-only order (close position only)")

	fs.Parse(os.Args[3:])

	// Validate required fields
	if *instrument == "" || *side == "" || *size == "" {
		fmt.Fprintf(os.Stderr, "Error: --instrument, --side, and --size are required\n\n")
		fs.PrintDefaults()
		os.Exit(1)
	}

	if (*orderType == "limit" || *orderType == "post_only") && *price == "" {
		fmt.Fprintf(os.Stderr, "Error: --price is required for limit/post_only orders\n\n")
		fs.PrintDefaults()
		os.Exit(1)
	}

	// Load config
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger (quiet mode for CLI)
	log := logger.NewTestLogger()

	// Initialize database
	db, err := storage.New(
		cfg.Database.Path,
		cfg.Database.WALMode,
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Initialize OKX client
	okxClient := okx.New(
		cfg.OKX.APIURL,
		cfg.OKX.APIKey,
		cfg.OKX.APISecret,
		cfg.OKX.Passphrase,
		cfg.OKX.Timeout,
		cfg.OKX.MaxRetries,
		false, // Disable debug for CLI
	)

	// Initialize notifier
	logNotifier := notifier.NewLogNotifier(log)

	// Initialize Order Control service
	orderControlService := ordercontrol.New(
		&cfg.OrderControl,
		okxClient,
		db,
		logNotifier,
		log,
	)

	// Create order request
	req := &okx.OrderRequest{
		InstId:     *instrument,
		TdMode:     "cross", // Default to cross margin
		Side:       *side,
		OrdType:    *orderType,
		Sz:         *size,
		Px:         *price,
		ReduceOnly: *reduceOnly,
	}

	fmt.Printf("Placing order...\n")
	fmt.Printf("  Instrument: %s\n", *instrument)
	fmt.Printf("  Side:       %s\n", *side)
	fmt.Printf("  Type:       %s\n", *orderType)
	fmt.Printf("  Size:       %s\n", *size)
	if *price != "" {
		fmt.Printf("  Price:      %s\n", *price)
	}
	fmt.Printf("  Reduce-only: %v\n\n", *reduceOnly)

	// Place order through Order Control
	ctx := context.Background()
	resp, err := orderControlService.PlaceOrder(ctx, req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ Order failed: %v\n", err)
		os.Exit(1)
	}

	// Check response
	if resp.Code != "0" {
		fmt.Fprintf(os.Stderr, "✗ Order rejected by OKX: %s - %s\n", resp.Code, resp.Msg)
		os.Exit(1)
	}

	// Success
	if len(resp.Data) > 0 {
		fmt.Printf("✓ Order placed successfully!\n")
		fmt.Printf("  Order ID: %s\n", resp.Data[0].OrdId)
		if resp.Data[0].SCode != "" && resp.Data[0].SCode != "0" {
			fmt.Printf("  Warning: %s - %s\n", resp.Data[0].SCode, resp.Data[0].SMsg)
		}
	} else {
		fmt.Printf("✓ Order submitted\n")
	}
}

func handleOrderStats() {
	// Load config
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize database
	db, err := storage.New(
		cfg.Database.Path,
		cfg.Database.WALMode,
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// Get current week start
	now := time.Now().UTC()
	weekStart := models.GetWeekStart(now)

	ctx := context.Background()

	// Get weekly stats
	stats, err := db.GetWeeklyOrderStats(ctx, weekStart)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get weekly stats: %v\n", err)
		os.Exit(1)
	}

	// Display stats
	fmt.Printf("Order Statistics for Week Starting %s\n", weekStart.Format("2006-01-02"))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if stats == nil || stats.TotalOrders == 0 {
		fmt.Printf("No orders placed this week.\n")
		return
	}

	fmt.Printf("Total Orders:        %d\n", stats.TotalOrders)
	fmt.Printf("Regular Orders:      %d\n", stats.RegularOrders)
	fmt.Printf("Reduce-Only Orders:  %d\n", stats.ReduceOnlyOrders)
	fmt.Printf("\nBy Status:\n")
	fmt.Printf("  Placed:   %d\n", stats.PlacedOrders)
	fmt.Printf("  Filled:   %d\n", stats.FilledOrders)
	fmt.Printf("  Canceled: %d\n", stats.CanceledOrders)
	fmt.Printf("  Failed:   %d\n", stats.FailedOrders)

	// Show limit info if configured
	if cfg.OrderControl.FrequencyLimit.Enabled {
		maxOrders := cfg.OrderControl.FrequencyLimit.WeeklyMaxOrders
		countToCheck := stats.TotalOrders
		if cfg.OrderControl.FrequencyLimit.ExcludeReduceOnly {
			countToCheck = stats.RegularOrders
		}

		remaining := maxOrders - countToCheck
		if remaining < 0 {
			remaining = 0
		}

		fmt.Printf("\nFrequency Limit:\n")
		fmt.Printf("  Used:      %d / %d", countToCheck, maxOrders)
		if cfg.OrderControl.FrequencyLimit.ExcludeReduceOnly {
			fmt.Printf(" (regular orders only)")
		}
		fmt.Printf("\n")
		fmt.Printf("  Remaining: %d\n", remaining)

		if remaining == 0 {
			fmt.Printf("  ⚠️  Weekly limit reached!\n")
		} else if remaining <= 3 {
			fmt.Printf("  ⚠️  Only %d orders remaining this week\n", remaining)
		}
	}
}

func handleOrderList() {
	fs := flag.NewFlagSet("order list", flag.ExitOnError)
	limit := fs.Int("limit", 10, "Number of orders to display")
	sync := fs.Bool("sync", true, "Sync orders from OKX API before displaying")
	debug := fs.Bool("debug", false, "Enable debug output for troubleshooting")
	fs.Parse(os.Args[3:])

	// Load config
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize database
	if *debug {
		fmt.Fprintf(os.Stderr, "DEBUG: Initializing database\n")
		fmt.Fprintf(os.Stderr, "  Path: %s\n", cfg.Database.Path)
		fmt.Fprintf(os.Stderr, "  WAL Mode: %v\n", cfg.Database.WALMode)
		fmt.Fprintf(os.Stderr, "  Max Open Conns: %d\n", cfg.Database.MaxOpenConns)
		fmt.Fprintf(os.Stderr, "  Max Idle Conns: %d\n", cfg.Database.MaxIdleConns)
	}

	db, err := storage.New(
		cfg.Database.Path,
		cfg.Database.WALMode,
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	if *debug {
		fmt.Fprintf(os.Stderr, "DEBUG: Database initialized successfully\n")
	}

	ctx := context.Background()

	// Sync orders from OKX API if requested
	if *sync {
		fmt.Printf("Syncing orders from OKX API...\n")

		// Initialize OKX client
		okxClient := okx.New(
			cfg.OKX.APIURL,
			cfg.OKX.APIKey,
			cfg.OKX.APISecret,
			cfg.OKX.Passphrase,
			cfg.OKX.Timeout,
			cfg.OKX.MaxRetries,
			false, // Disable debug for CLI
		)

		// Fetch order history from OKX (SWAP orders, last 7 days)
		histResp, err := okxClient.GetOrdersHistory(ctx, "SWAP", 100)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch order history from OKX: %v\n", err)
			fmt.Fprintf(os.Stderr, "Showing local orders only...\n\n")
		} else {
			// Sync orders to local database
			syncedCount := 0
			for _, okxOrder := range histResp.Data {
				// Skip canceled orders - they don't contribute to trading analysis
				// Only record filled and partially_filled orders for statistics
				if okxOrder.State == "canceled" || okxOrder.State == "live" {
					continue
				}

				// Parse timestamps (OKX returns milliseconds as string)
				var placedAtMs time.Time
				if ms, err := strconv.ParseInt(okxOrder.CTime, 10, 64); err == nil {
					placedAtMs = time.Unix(0, ms*1000000) // Convert ms to ns
				} else {
					// Fallback: try parsing as seconds
					if sec, err := strconv.ParseInt(okxOrder.CTime, 10, 64); err == nil {
						placedAtMs = time.Unix(sec, 0)
					} else {
						fmt.Fprintf(os.Stderr, "Warning: Failed to parse timestamp for order %s: %v\n", okxOrder.OrdId, err)
						continue
					}
				}

				reduceOnly := okxOrder.ReduceOnly == "true"
				weekStart := models.GetWeekStart(placedAtMs)

				orderHistory := &models.OrderHistory{
					OrderID:    okxOrder.OrdId,
					InstId:     okxOrder.InstId,
					Side:       okxOrder.Side,
					OrdType:    okxOrder.OrdType,
					Size:       okxOrder.Sz,
					Price:      okxOrder.Px,
					ReduceOnly: reduceOnly,
					PlacedAt:   placedAtMs,
					WeekStart:  weekStart,
					Status:     okxOrder.State,
					CreatedAt:  time.Now().UTC(),
				}

				// Insert into database (duplicate check is handled by InsertOrderHistory)
				if err := db.InsertOrderHistory(ctx, orderHistory); err != nil {
					if *debug {
						fmt.Fprintf(os.Stderr, "DEBUG: Failed to sync order %s\n", okxOrder.OrdId)
						fmt.Fprintf(os.Stderr, "  Error: %v\n", err)
						fmt.Fprintf(os.Stderr, "  InstId: %s, Side: %s, Size: %s\n", okxOrder.InstId, okxOrder.Side, okxOrder.Sz)
					} else {
						fmt.Fprintf(os.Stderr, "Warning: Failed to sync order %s: %v\n", okxOrder.OrdId, err)
					}
				} else {
					syncedCount++
					if *debug {
						fmt.Fprintf(os.Stderr, "DEBUG: Successfully synced order %s\n", okxOrder.OrdId)
					}
				}
			}

			fmt.Printf("Synced %d orders from OKX API\n\n", syncedCount)
		}
	}

	// Get current week start
	now := time.Now().UTC()
	weekStart := models.GetWeekStart(now)

	// Get orders for current week from local database
	orders, err := db.GetOrdersForWeek(ctx, weekStart)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get orders: %v\n", err)
		os.Exit(1)
	}

	// Display orders
	fmt.Printf("Recent Orders (Week Starting %s)\n", weekStart.Format("2006-01-02"))
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if len(orders) == 0 {
		fmt.Printf("No orders found for this week.\n")
		fmt.Printf("\nTip: Use --sync=true to fetch orders from OKX API\n")
		return
	}

	// Limit number of displayed orders
	displayCount := len(orders)
	if *limit > 0 && *limit < displayCount {
		displayCount = *limit
	}

	// Use tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TIME\tINSTRUMENT\tSIDE\tTYPE\tSIZE\tPRICE\tSTATUS")
	fmt.Fprintln(w, "────\t──────────\t────\t────\t────\t─────\t──────")

	// Display most recent first
	for i := len(orders) - 1; i >= len(orders)-displayCount; i-- {
		order := orders[i]
		timeStr := order.PlacedAt.Format("01-02 15:04")
		priceStr := order.Price
		if priceStr == "" {
			priceStr = "market"
		}

		reduceOnlyMark := ""
		if order.ReduceOnly {
			reduceOnlyMark = " ®"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s%s\n",
			timeStr,
			order.InstId,
			order.Side,
			order.OrdType,
			order.Size,
			priceStr,
			order.Status,
			reduceOnlyMark,
		)
	}
	w.Flush()

	if len(orders) > displayCount {
		fmt.Printf("\n(Showing %d of %d orders. Use --limit to show more)\n", displayCount, len(orders))
	}
}

func printUsage() {
	fmt.Println("TenyoJubaku CLI - Order Control Command Line Tool")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  tenyojubaku-cli <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  order place      Place a new order with frequency and maker-only checks")
	fmt.Println("  order stats      Show weekly order statistics")
	fmt.Println("  order list       List recent orders")
	fmt.Println("  position list    List closed positions history")
	fmt.Println("  help             Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Place a limit buy order")
	fmt.Println("  tenyojubaku-cli order place --instrument BTC-USDT-SWAP --side buy --size 0.01 --price 50000")
	fmt.Println()
	fmt.Println("  # Place a reduce-only sell order")
	fmt.Println("  tenyojubaku-cli order place --instrument BTC-USDT-SWAP --side sell --size 0.01 --price 51000 --reduce-only")
	fmt.Println()
	fmt.Println("  # View this week's statistics")
	fmt.Println("  tenyojubaku-cli order stats")
	fmt.Println()
	fmt.Println("  # List recent orders")
	fmt.Println("  tenyojubaku-cli order list --limit 20")
	fmt.Println()
	fmt.Println("  # List closed positions with sync")
	fmt.Println("  tenyojubaku-cli position list --sync=true --limit 30")
}

func printOrderUsage() {
	fmt.Println("Usage: tenyojubaku-cli order <subcommand> [options]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  place    Place a new order")
	fmt.Println("  stats    Show weekly statistics")
	fmt.Println("  list     List recent orders")
	fmt.Println()
	fmt.Println("Run 'tenyojubaku-cli order <subcommand> -h' for more details")
}

// ============================================================================
// Position Commands (V3.2)
// ============================================================================

func handlePositionCommand() {
	if len(os.Args) < 3 {
		printPositionUsage()
		os.Exit(1)
	}

	subcommand := os.Args[2]

	switch subcommand {
	case "list":
		handlePositionList()
	default:
		fmt.Fprintf(os.Stderr, "Unknown position subcommand: %s\n\n", subcommand)
		printPositionUsage()
		os.Exit(1)
	}
}

func handlePositionList() {
	fs := flag.NewFlagSet("position list", flag.ExitOnError)

	sync := fs.Bool("sync", false, "Sync positions from OKX API before listing")
	limit := fs.Int("limit", 20, "Limit number of positions to display (0 = all)")
	instrument := fs.String("instrument", "", "Filter by instrument ID (e.g., BTC-USDT-SWAP)")

	fs.Parse(os.Args[3:])

	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize database
	db, err := storage.New(
		cfg.Database.Path,
		cfg.Database.WALMode,
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()

	// Sync positions from OKX API if requested
	if *sync {
		fmt.Printf("Syncing positions from OKX API...\n")

		// Initialize OKX client
		okxClient := okx.New(
			cfg.OKX.APIURL,
			cfg.OKX.APIKey,
			cfg.OKX.APISecret,
			cfg.OKX.Passphrase,
			cfg.OKX.Timeout,
			cfg.OKX.MaxRetries,
			false, // Disable debug for CLI
		)

		// Fetch position history from OKX (SWAP positions, last 3 months)
		histResp, err := okxClient.GetPositionsHistory(ctx, "SWAP", 100)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to fetch position history from OKX: %v\n", err)
			fmt.Fprintf(os.Stderr, "Showing local positions only...\n\n")
		} else {
			// Sync positions to local database
			syncedCount := 0
			for _, okxPos := range histResp.Data {
				// Parse timestamps (OKX returns milliseconds as string)
				var openedAt, closedAt time.Time
				if ms, err := strconv.ParseInt(okxPos.CTime, 10, 64); err == nil {
					openedAt = time.Unix(0, ms*1000000) // Convert ms to ns
				} else {
					fmt.Fprintf(os.Stderr, "Warning: Failed to parse CTime for position %s: %v\n", okxPos.PosId, err)
					continue
				}

				if ms, err := strconv.ParseInt(okxPos.UTime, 10, 64); err == nil {
					closedAt = time.Unix(0, ms*1000000)
				} else {
					fmt.Fprintf(os.Stderr, "Warning: Failed to parse UTime for position %s: %v\n", okxPos.PosId, err)
					continue
				}

				positionHistory := &models.PositionHistory{
					PosId:         okxPos.PosId,
					InstType:      okxPos.InstType,
					InstId:        okxPos.InstId,
					MgnMode:       okxPos.MgnMode,
					PosSide:       okxPos.PosSide,
					Lever:         okxPos.Lever,
					OpenAvgPx:     okxPos.OpenAvgPx,
					CloseAvgPx:    okxPos.CloseAvgPx,
					OpenMaxPos:    okxPos.OpenMaxPos,
					CloseTotalPos: okxPos.CloseTotalPos,
					RealizedPnl:   okxPos.RealizedPnl,
					Pnl:           okxPos.Pnl,
					PnlRatio:      okxPos.PnlRatio,
					Fee:           okxPos.Fee,
					FundingFee:    okxPos.FundingFee,
					LiqPenalty:    okxPos.LiqPenalty,
					CloseType:     okxPos.Type,
					Direction:     okxPos.Direction,
					Ccy:           okxPos.Ccy,
					OpenedAt:      openedAt,
					ClosedAt:      closedAt,
					CreatedAt:     time.Now().UTC(),
				}

				// Insert into database (duplicate check is handled by InsertPositionHistory)
				if err := db.InsertPositionHistory(ctx, positionHistory); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: Failed to sync position %s: %v\n", okxPos.PosId, err)
				} else {
					syncedCount++
				}
			}

			fmt.Printf("Synced %d positions from OKX API\n\n", syncedCount)
		}
	}

	// Get positions from local database
	displayLimit := 0
	if *limit > 0 {
		displayLimit = *limit
	}

	positions, err := db.GetPositionsHistory(ctx, *instrument, displayLimit)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get positions: %v\n", err)
		os.Exit(1)
	}

	// Display positions
	if *instrument != "" {
		fmt.Printf("Position History for %s\n", *instrument)
	} else {
		fmt.Printf("Position History (Recent %d positions)\n", len(positions))
	}
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	if len(positions) == 0 {
		fmt.Printf("No positions found.\n")
		fmt.Printf("\nTip: Use --sync=true to fetch positions from OKX API\n")
		return
	}

	// Use tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CLOSED AT\tINSTRUMENT\tSIDE\tSIZE\tOPEN PX\tCLOSE PX\tPNL\tCLOSE TYPE")
	fmt.Fprintln(w, "─────────\t──────────\t────\t────\t───────\t────────\t───\t──────────")

	for _, pos := range positions {
		timeStr := pos.ClosedAt.Format("01-02 15:04")
		closeTypeDesc := pos.GetCloseTypeDescription()

		// Format PNL with + or - sign
		pnlStr := pos.RealizedPnl
		if pnlStr != "" && pnlStr != "0" && pnlStr[0] != '-' {
			pnlStr = "+" + pnlStr
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			timeStr,
			pos.InstId,
			pos.PosSide,
			pos.CloseTotalPos,
			pos.OpenAvgPx,
			pos.CloseAvgPx,
			pnlStr,
			closeTypeDesc,
		)
	}
	w.Flush()

	fmt.Printf("\nTotal Positions: %d\n", len(positions))
}

func printPositionUsage() {
	fmt.Println("Usage: tenyojubaku-cli position <subcommand> [options]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  list     List closed positions history")
	fmt.Println()
	fmt.Println("Run 'tenyojubaku-cli position <subcommand> -h' for more details")
}
