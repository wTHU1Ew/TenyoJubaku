package notifier

import (
	"context"
	"sync"
)

// MockNotifier is a mock implementation of Interface for testing.
// It tracks all confirmation requests sent and allows configuring error behavior.
type MockNotifier struct {
	mu sync.Mutex

	// Requests stores all confirmation requests that were sent
	Requests []ConfirmationInfo

	// SendError is the error to return from SendConfirmationRequest
	// Set this to simulate notification failures
	SendError error

	// CallCount tracks the total number of calls to SendConfirmationRequest
	CallCount int
}

// NewMockNotifier creates a new MockNotifier instance.
func NewMockNotifier() *MockNotifier {
	return &MockNotifier{
		Requests: make([]ConfirmationInfo, 0),
	}
}

// SendConfirmationRequest records the request and returns the configured error.
func (m *MockNotifier) SendConfirmationRequest(ctx context.Context, info ConfirmationInfo) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.CallCount++
	m.Requests = append(m.Requests, info)

	return m.SendError
}

// Reset clears all recorded requests and resets the call count.
// Useful for reusing the same mock across multiple test cases.
func (m *MockNotifier) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Requests = make([]ConfirmationInfo, 0)
	m.CallCount = 0
	m.SendError = nil
}

// GetRequests returns a copy of all recorded requests.
// Thread-safe for use in test assertions.
func (m *MockNotifier) GetRequests() []ConfirmationInfo {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make([]ConfirmationInfo, len(m.Requests))
	copy(result, m.Requests)
	return result
}

// GetCallCount returns the total number of calls to SendConfirmationRequest.
func (m *MockNotifier) GetCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.CallCount
}

// Ensure MockNotifier implements Interface at compile time
var _ Interface = (*MockNotifier)(nil)
