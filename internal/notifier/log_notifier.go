package notifier

import (
	"context"

	"github.com/wTHU1Ew/TenyoJubaku/internal/logger"
)

// LogNotifier implements the Interface by logging confirmation requests.
// This is the default implementation suitable for console-based workflows.
type LogNotifier struct {
	logger *logger.Logger
}

// NewLogNotifier creates a new LogNotifier instance.
func NewLogNotifier(logger *logger.Logger) *LogNotifier {
	return &LogNotifier{logger: logger}
}

// SendConfirmationRequest logs a confirmation request with prominent formatting.
func (n *LogNotifier) SendConfirmationRequest(ctx context.Context, info ConfirmationInfo) error {
	// Log a prominent warning message
	n.logger.Warn("==========================================")
	n.logger.Warn("   CONFIRMATION REQUIRED")
	n.logger.Warn("==========================================")
	n.logger.Warn("Order ID:     %s", info.OrderID)
	n.logger.Warn("Instrument:   %s", info.Instrument)
	n.logger.Warn("Side:         %s", info.Side)
	n.logger.Warn("Size:         %s", info.Size)
	if info.Price != "" {
		n.logger.Warn("Price:        %s", info.Price)
	}
	n.logger.Warn("Placed At:    %s", info.PlacedAt.Format("2006-01-02 15:04:05 MST"))
	n.logger.Warn("Timeout In:   %v", info.TimeoutIn)
	n.logger.Warn("Confirmations: %d", info.ConfirmationCount)
	n.logger.Warn("Timeouts:     %d / %d", info.TimeoutCount, info.MaxTimeouts)
	n.logger.Warn("")
	n.logger.Warn("Please confirm this order to prevent size reduction.")
	n.logger.Warn("If not confirmed within %v, the size will be reduced.", info.TimeoutIn)
	if info.TimeoutCount >= info.MaxTimeouts-1 {
		n.logger.Warn("WARNING: This is the last timeout before cancellation!")
	}
	n.logger.Warn("==========================================")

	return nil
}

// Ensure LogNotifier implements Interface at compile time
var _ Interface = (*LogNotifier)(nil)
