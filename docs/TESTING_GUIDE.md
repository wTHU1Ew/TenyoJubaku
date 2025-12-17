# Testing Guide - Using Mocks

This guide demonstrates how to write tests for TenyoJubaku services using the new interface-based architecture.

## Quick Start

### Basic Test with Mocks

```go
package mypackage

import (
    "testing"

    "github.com/wTHU1Ew/TenyoJubaku/internal/okx"
    "github.com/wTHU1Ew/TenyoJubaku/internal/storage"
    "github.com/wTHU1Ew/TenyoJubaku/internal/monitor"
    "github.com/wTHU1Ew/TenyoJubaku/internal/logger"
)

func TestMyFeature(t *testing.T) {
    // 1. Create mock dependencies
    mockStorage := &storage.MockStorage{}
    mockOKX := &okx.MockClient{}
    testLogger, _ := logger.New("", 0, 10, 7, 3, false, false)
    defer testLogger.Close()

    // 2. Create service with mocks
    mon := monitor.New(mockOKX, mockStorage, testLogger, 60)

    // 3. Test your feature
    err := mon.HealthCheck()

    // 4. Assert results
    if err != nil {
        t.Errorf("Expected no error, got: %v", err)
    }

    // 5. Verify interactions
    if mockOKX.HealthCheckCallCount != 1 {
        t.Errorf("Expected 1 health check call, got %d", mockOKX.HealthCheckCallCount)
    }
}
```

## Common Testing Patterns

### Pattern 1: Testing Happy Path

```go
func TestMonitor_FetchBalances_Success(t *testing.T) {
    mockStorage := &storage.MockStorage{
        Balances: make([]models.AccountBalance, 0),
    }

    mockOKX := &okx.MockClient{
        // Mock returns successful balance response
        GetAccountBalanceFunc: func() (*okx.AccountBalanceResponse, error) {
            // Return test data
            resp := &okx.AccountBalanceResponse{Code: "0"}
            // ... populate with test data
            return resp, nil
        },
    }

    testLogger, _ := logger.New("", 0, 10, 7, 3, false, false)
    defer testLogger.Close()

    mon := monitor.New(mockOKX, mockStorage, testLogger, 60)

    // Execute
    err := mon.fetchAndStoreBalances()

    // Verify
    if err != nil {
        t.Errorf("Expected success, got error: %v", err)
    }

    if len(mockStorage.Balances) == 0 {
        t.Error("Expected balances to be stored")
    }
}
```

### Pattern 2: Testing Error Handling

```go
func TestMonitor_StorageFailure(t *testing.T) {
    // Mock storage that always fails
    mockStorage := &storage.MockStorage{
        InsertBalanceFunc: func(b *models.AccountBalance) error {
            return fmt.Errorf("database connection lost")
        },
    }

    mockOKX := &okx.MockClient{} // Default behavior
    testLogger, _ := logger.New("", 0, 10, 7, 3, false, false)
    defer testLogger.Close()

    mon := monitor.New(mockOKX, mockStorage, testLogger, 60)

    // Should handle error gracefully
    err := mon.fetchAndStoreBalances()

    if err == nil {
        t.Error("Expected error, got nil")
    }
}
```

### Pattern 3: Testing State Changes

```go
func TestMonitor_CallCounting(t *testing.T) {
    mockStorage := &storage.MockStorage{}
    mockOKX := &okx.MockClient{}
    testLogger, _ := logger.New("", 0, 10, 7, 3, false, false)
    defer testLogger.Close()

    mon := monitor.New(mockOKX, mockStorage, testLogger, 60)

    // Call multiple times
    mon.fetchAndStore()
    mon.fetchAndStore()
    mon.fetchAndStore()

    // Verify call counts
    expectedCalls := 3
    if mockOKX.GetAccountBalanceCallCount != expectedCalls {
        t.Errorf("Expected %d balance calls, got %d",
            expectedCalls, mockOKX.GetAccountBalanceCallCount)
    }

    if mockOKX.GetPositionsCallCount != expectedCalls {
        t.Errorf("Expected %d position calls, got %d",
            expectedCalls, mockOKX.GetPositionsCallCount)
    }
}
```

### Pattern 4: Table-Driven Tests

```go
func TestMonitor_DifferentScenarios(t *testing.T) {
    tests := []struct {
        name          string
        setupOKX      func() *okx.MockClient
        setupStorage  func() *storage.MockStorage
        expectError   bool
    }{
        {
            name: "Successful fetch",
            setupOKX: func() *okx.MockClient {
                return &okx.MockClient{} // Default behavior
            },
            setupStorage: func() *storage.MockStorage {
                return &storage.MockStorage{}
            },
            expectError: false,
        },
        {
            name: "OKX API failure",
            setupOKX: func() *okx.MockClient {
                return &okx.MockClient{
                    GetPositionsFunc: func() (*okx.PositionsResponse, error) {
                        return nil, fmt.Errorf("API timeout")
                    },
                }
            },
            setupStorage: func() *storage.MockStorage {
                return &storage.MockStorage{}
            },
            expectError: true,
        },
        {
            name: "Storage write failure",
            setupOKX: func() *okx.MockClient {
                return &okx.MockClient{}
            },
            setupStorage: func() *storage.MockStorage {
                return &storage.MockStorage{
                    InsertPositionFunc: func(p *models.Position) error {
                        return fmt.Errorf("disk full")
                    },
                }
            },
            expectError: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockOKX := tt.setupOKX()
            mockStorage := tt.setupStorage()
            testLogger, _ := logger.New("", 0, 10, 7, 3, false, false)
            defer testLogger.Close()

            mon := monitor.New(mockOKX, mockStorage, testLogger, 60)

            err := mon.fetchAndStore()

            if tt.expectError && err == nil {
                t.Error("Expected error but got none")
            }
            if !tt.expectError && err != nil {
                t.Errorf("Expected no error but got: %v", err)
            }
        })
    }
}
```

### Pattern 5: Verifying Arguments

```go
func TestTPSL_CorrectOrderPlacement(t *testing.T) {
    mockOKX := &okx.MockClient{}
    testLogger, _ := logger.New("", 0, 10, 7, 3, false, false)
    defer testLogger.Close()

    manager := tpsl.New(config, mockOKX, testLogger)

    position := &models.Position{
        Instrument:   "BTC-USDT-SWAP",
        PositionSide: models.PositionSideLong,
        PositionSize: 1.5,
        AveragePrice: 50000,
    }

    // Execute
    manager.placeTPSLOrder(position, 1.5, prices)

    // Verify the order request
    if mockOKX.LastAlgoOrderRequest == nil {
        t.Fatal("Expected algo order to be placed")
    }

    req := mockOKX.LastAlgoOrderRequest
    if req.InstId != "BTC-USDT-SWAP" {
        t.Errorf("Expected instrument BTC-USDT-SWAP, got %s", req.InstId)
    }

    if req.Side != "sell" { // Long position closes with sell
        t.Errorf("Expected side 'sell', got %s", req.Side)
    }
}
```

## Mock Configuration

### Storage Mock Options

```go
// 1. Default behavior (in-memory storage)
mockStorage := &storage.MockStorage{}

// 2. Pre-populated data
mockStorage := &storage.MockStorage{
    Positions: []models.Position{
        {Instrument: "BTC-USDT-SWAP", PositionSize: 1.5},
    },
    Balances: []models.AccountBalance{
        {Currency: "USDT", Balance: 10000},
    },
}

// 3. Custom behavior
mockStorage := &storage.MockStorage{
    InsertPositionFunc: func(p *models.Position) error {
        // Custom logic
        if p.PositionSize < 0 {
            return fmt.Errorf("invalid size")
        }
        return nil
    },
}

// 4. Simulating failures
mockStorage := &storage.MockStorage{
    HealthCheckFunc: func() error {
        return fmt.Errorf("database down")
    },
}
```

### OKX Client Mock Options

```go
// 1. Default behavior (successful responses)
mockOKX := &okx.MockClient{}

// 2. Custom responses
mockOKX := &okx.MockClient{
    GetPositionsFunc: func() (*okx.PositionsResponse, error) {
        return &okx.PositionsResponse{
            Code: "0",
            Data: []okx.PositionData{
                {InstId: "BTC-USDT-SWAP", Pos: "1.5"},
            },
        }, nil
    },
}

// 3. Simulating API errors
mockOKX := &okx.MockClient{
    GetAccountBalanceFunc: func() (*okx.AccountBalanceResponse, error) {
        return nil, fmt.Errorf("rate limited")
    },
}

// 4. Tracking calls
mockOKX := &okx.MockClient{}
// ... use mockOKX ...
fmt.Printf("GetPositions called %d times\n", mockOKX.GetPositionsCallCount)
fmt.Printf("Last ticker requested: %s\n", mockOKX.LastTickerInstId)
```

## Best Practices

### ✅ DO

1. **Use mocks for unit tests**
   ```go
   // Unit test - fast, isolated
   func TestBusinessLogic(t *testing.T) {
       mockStorage := &storage.MockStorage{}
       mockOKX := &okx.MockClient{}
       // Test business logic
   }
   ```

2. **Use real implementations for integration tests**
   ```go
   // Integration test - slower, comprehensive
   func TestIntegration(t *testing.T) {
       if testing.Short() {
           t.Skip("Skipping integration test")
       }
       db, _ := storage.New("/tmp/test.db", true, 10, 5)
       defer db.Close()
       // Test full system
   }
   ```

3. **Reset mocks between tests**
   ```go
   func TestMultiple(t *testing.T) {
       mockOKX := &okx.MockClient{}

       // Test 1
       mockOKX.Reset() // Clear counters

       // Test 2
       mockOKX.Reset() // Clear again
   }
   ```

4. **Verify interactions**
   ```go
   if mockOKX.PlaceAlgoOrderCallCount != expectedOrders {
       t.Errorf("Expected %d orders, placed %d",
           expectedOrders, mockOKX.PlaceAlgoOrderCallCount)
   }
   ```

### ❌ DON'T

1. **Don't use real APIs in unit tests**
   ```go
   // ❌ Bad - slow, flaky, costs money
   func TestLogic(t *testing.T) {
       okxClient := okx.New(realAPIKey, ...)
       // ...
   }

   // ✅ Good - fast, reliable, free
   func TestLogic(t *testing.T) {
       mockOKX := &okx.MockClient{}
       // ...
   }
   ```

2. **Don't use real database in unit tests**
   ```go
   // ❌ Bad - slow, requires setup
   func TestLogic(t *testing.T) {
       db, _ := storage.New("/tmp/test.db", ...)
       defer db.Close()
       // ...
   }

   // ✅ Good - fast, no setup
   func TestLogic(t *testing.T) {
       mockStorage := &storage.MockStorage{}
       // ...
   }
   ```

3. **Don't forget to clean up**
   ```go
   // ✅ Always defer cleanup
   testLogger, _ := logger.New(...)
   defer testLogger.Close()
   ```

## Running Tests

```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/monitor

# Run with verbose output
go test -v ./internal/monitor

# Run specific test
go test -v ./internal/monitor -run TestMonitor_WithMocks

# Run tests with coverage
go test -cover ./internal/monitor

# Skip slow integration tests
go test -short ./...

# Run benchmarks
go test -bench=. ./internal/monitor
```

## Example: Complete Test File

See `internal/monitor/monitor_test.go` for a complete example demonstrating:
- Basic mock usage
- Error scenario testing
- Performance benchmarking
- Table-driven tests
- Proper setup and teardown

## Additional Resources

- [INTERFACE_REFACTORING.md](../INTERFACE_REFACTORING.md) - Architecture changes
- [ARCHITECTURE_REVIEW.md](../ARCHITECTURE_REVIEW.md) - Original architecture review
- Go testing package: https://pkg.go.dev/testing
- Table-driven tests: https://go.dev/wiki/TableDrivenTests
