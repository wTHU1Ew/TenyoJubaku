package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wTHU1Ew/TenyoJubaku/internal/config"
	"github.com/wTHU1Ew/TenyoJubaku/internal/logger"
	"github.com/wTHU1Ew/TenyoJubaku/internal/monitor"
	"github.com/wTHU1Ew/TenyoJubaku/internal/notifier"
	"github.com/wTHU1Ew/TenyoJubaku/internal/okx"
	"github.com/wTHU1Ew/TenyoJubaku/internal/ordercontrol"
	"github.com/wTHU1Ew/TenyoJubaku/internal/storage"
	"github.com/wTHU1Ew/TenyoJubaku/internal/tpsl"
	"github.com/wTHU1Ew/TenyoJubaku/internal/version"
)

func main() {
	// Exit code
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	// Load configuration
	cfg, err := config.Load("configs/config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		exitCode = 1
		return
	}

	// Parse log level
	logLevel, err := logger.ParseLevel(cfg.Logging.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse log level: %v\n", err)
		exitCode = 1
		return
	}

	// Initialize logger
	log, err := logger.New(
		cfg.Logging.FilePath,
		logLevel,
		cfg.Logging.MaxSize,
		cfg.Logging.MaxAge,
		cfg.Logging.MaxBackups,
		cfg.Logging.Compress,
		cfg.Logging.Console,
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		exitCode = 1
		return
	}
	defer log.Close()

	log.Info("=== TenyoJubaku Starting ===")
	log.Info("Version: %s", version.GetFullVersion())
	log.Info("Configuration loaded: %s", cfg.MaskSensitive())

	// Initialize database
	log.Info("Initializing database at %s", cfg.Database.Path)
	db, err := storage.New(
		cfg.Database.Path,
		cfg.Database.WALMode,
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
	)
	if err != nil {
		log.Error("Failed to initialize database: %v", err)
		exitCode = 1
		return
	}
	defer db.Close()
	log.Info("Database initialized successfully")

	// Initialize OKX API client
	log.Info("Initializing OKX API client")
	if cfg.OKX.DebugEnable {
		log.Info("OKX API debug mode enabled - API responses will be printed to console")
	}
	okxClient := okx.New(
		cfg.OKX.APIURL,
		cfg.OKX.APIKey,
		cfg.OKX.APISecret,
		cfg.OKX.Passphrase,
		cfg.OKX.Timeout,
		cfg.OKX.MaxRetries,
		cfg.OKX.DebugEnable,
	)

	// Initialize monitoring service
	log.Info("Initializing monitoring service")
	monitorService := monitor.New(
		okxClient,
		db,
		log,
		cfg.Monitoring.Interval,
	)

	// Initialize notifier (for Order Control alerts)
	log.Info("Initializing notifier")
	logNotifier := notifier.NewLogNotifier(log)

	// Initialize Order Control service if enabled
	// Note: Order Control service is not a background service, it's used by other components
	// when placing orders. For now, we just initialize it and log its configuration.
	// TODO: Integrate Order Control with TPSL or other order-placing components
	if cfg.OrderControl.Enabled {
		log.Info("Initializing Order Control service")
		log.Info("Frequency limit: enabled=%v, weekly_max_orders=%d, exclude_reduce_only=%v",
			cfg.OrderControl.FrequencyLimit.Enabled,
			cfg.OrderControl.FrequencyLimit.WeeklyMaxOrders,
			cfg.OrderControl.FrequencyLimit.ExcludeReduceOnly)
		log.Info("Maker-only: enabled=%v, min_price_distance_pct=%.2f%%",
			cfg.OrderControl.MakerOnly.Enabled,
			cfg.OrderControl.MakerOnly.MinPriceDistancePct)

		_ = ordercontrol.New(
			&cfg.OrderControl,
			okxClient,
			db,
			logNotifier,
			log,
		)
		log.Info("Order Control service initialized (ready for integration)")
	} else {
		log.Info("Order Control disabled in configuration")
	}

	// Initialize TPSL scheduler if enabled
	var tpslScheduler *tpsl.Scheduler
	if cfg.TPSL.Enabled {
		log.Info("Initializing TPSL scheduler (real-time API mode)")
		// Pass dynamic SL config and storage for dynamic trailing stop-loss functionality
		tpslScheduler = tpsl.NewScheduler(&cfg.TPSL, &cfg.DynamicSL, okxClient, db, log)
		if cfg.DynamicSL.Enabled {
			log.Info("Dynamic trailing stop-loss enabled: firstMove=%.2f%%, trailingStep=%.2f%%, stopMoveStep=%.2f%%",
				cfg.DynamicSL.FirstMovePct*100, cfg.DynamicSL.TrailingStepPct*100, cfg.DynamicSL.StopMoveStepPct*100)
		}
	} else {
		log.Info("TPSL management disabled in configuration")
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start monitoring service in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := monitorService.Start(); err != nil {
			errChan <- err
		}
	}()

	// Start TPSL scheduler if enabled
	if tpslScheduler != nil {
		tpslScheduler.Start()
	}

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		log.Info("Received signal: %v", sig)
		log.Info("Initiating graceful shutdown...")

		// Stop TPSL scheduler if running
		if tpslScheduler != nil {
			tpslScheduler.Stop()
		}

		// Stop monitoring service
		monitorService.Stop()

		// Get final metrics
		metrics := monitorService.GetMetrics()
		log.Info("Final metrics: success_count=%v, error_count=%v, last_success=%v",
			metrics["success_count"], metrics["error_count"], metrics["last_success"])

		log.Info("Shutdown complete")

	case err := <-errChan:
		log.Error("Monitoring service error: %v", err)
		exitCode = 1
	}

	log.Info("=== TenyoJubaku Stopped ===")
}
