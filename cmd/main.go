package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/wTHU1Ew/TenyoJubaku/internal/config"
	"github.com/wTHU1Ew/TenyoJubaku/internal/logger"
	"github.com/wTHU1Ew/TenyoJubaku/internal/monitor"
	"github.com/wTHU1Ew/TenyoJubaku/internal/okx"
	"github.com/wTHU1Ew/TenyoJubaku/internal/storage"
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
	okxClient := okx.New(
		cfg.OKX.APIURL,
		cfg.OKX.APIKey,
		cfg.OKX.APISecret,
		cfg.OKX.Passphrase,
		cfg.OKX.Timeout,
		cfg.OKX.MaxRetries,
	)

	// Initialize monitoring service
	log.Info("Initializing monitoring service")
	monitorService := monitor.New(
		okxClient,
		db,
		log,
		cfg.Monitoring.Interval,
	)

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

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		log.Info("Received signal: %v", sig)
		log.Info("Initiating graceful shutdown...")

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
