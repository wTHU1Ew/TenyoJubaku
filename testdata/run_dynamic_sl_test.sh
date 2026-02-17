#!/bin/bash
# Dynamic SL V5.1 Integration Test Script
# Usage: ./run_dynamic_sl_test.sh <test_number> <price_file> [volatility_pct]
# Example: ./run_dynamic_sl_test.sh 1 dynamic_sl_test1_prices.txt 0.01

TEST_NUM=$1
PRICE_FILE="testdata/$2"
VOLATILITY=${3:-0.01}

SIM_URL="http://localhost:8888"
BIN_DIR="./bin"
LOG_DIR="/tmp/dynsl_test${TEST_NUM}"
SIM_INTERVAL=0.167  # 10 seconds per price step

mkdir -p "$LOG_DIR"

echo "=========================================="
echo "Dynamic SL Test $TEST_NUM"
echo "Price file: $PRICE_FILE"
echo "Volatility: $VOLATILITY (initial SL distance)"
echo "Simulator interval: ${SIM_INTERVAL}min (10s)"
echo "TPSL check_interval: 20s"
echo "monitoring.interval: 5s"
echo "=========================================="

# Update volatility_pct in config
sed -i '' "s/volatility_pct: [0-9.]*/volatility_pct: $VOLATILITY/" configs/config.yaml
echo "Config: volatility_pct set to $VOLATILITY"

# Clean database for fresh test
rm -f data/tenyojubaku.db data/tenyojubaku.db-shm data/tenyojubaku.db-wal
echo "Database cleaned"

# Kill any lingering processes
pkill -f okx-simulator 2>/dev/null
pkill -f tenyojubaku 2>/dev/null
sleep 1

# Start simulator
echo ""
echo "[Step 1] Starting simulator..."
"$BIN_DIR/okx-simulator" -file="$PRICE_FILE" -interval=$SIM_INTERVAL > "$LOG_DIR/simulator.log" 2>&1 &
SIM_PID=$!
echo "Simulator PID: $SIM_PID"
sleep 2

# Verify simulator is up
TICKER=$(curl -s "$SIM_URL/api/v5/market/ticker?instId=BTC-USDT-SWAP")
INIT_PRICE=$(echo "$TICKER" | grep -o '"last":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Simulator price: $INIT_PRICE"

# Start TenyoJubaku
echo ""
echo "[Step 2] Starting TenyoJubaku..."
"$BIN_DIR/tenyojubaku" > "$LOG_DIR/tenyojubaku.log" 2>&1 &
TJ_PID=$!
echo "TenyoJubaku PID: $TJ_PID"
sleep 3

# Determine order direction
if [[ "$TEST_NUM" == "3" || "$TEST_NUM" == "4" ]]; then
    SIDE="sell"
    SIDE_LABEL="空头(short)"
else
    SIDE="buy"
    SIDE_LABEL="多头(long)"
fi

# Place order
echo ""
echo "[Step 3] Placing $SIDE_LABEL order (limit at 100)..."
ORDER_RESULT=$("$BIN_DIR/tenyojubaku-cli" order place \
    --instrument BTC-USDT-SWAP \
    --side $SIDE \
    --type limit \
    --price 100 \
    --size 0.01 2>&1)
echo "Order result: $ORDER_RESULT"

sleep 2

echo ""
echo "[Step 4] Monitoring events (watching logs every 5s)..."
echo "Events will be captured to $LOG_DIR/"
echo ""

# Monitor loop
START_TIME=$(date +%s)
LAST_SL=""
SL_TRIGGER_PRICE=""
INITIAL_SL_RECORDED=false
PHASE_A_RECORDED=false
PHASE_B_COUNT=0
TPSL_ALGO_ID=""

MAX_WAIT=600  # 10 minutes max
while true; do
    ELAPSED=$(( $(date +%s) - START_TIME ))
    if [ $ELAPSED -ge $MAX_WAIT ]; then
        echo "Timeout after ${MAX_WAIT}s"
        break
    fi

    # Get current price from simulator
    TICKER=$(curl -s "$SIM_URL/api/v5/market/ticker?instId=BTC-USDT-SWAP" 2>/dev/null)
    CURR_PRICE=$(echo "$TICKER" | grep -o '"last":"[^"]*"' | head -1 | cut -d'"' -f4)

    # Get current positions
    POSITIONS=$(curl -s "$SIM_URL/api/v5/account/positions?instId=BTC-USDT-SWAP" 2>/dev/null)
    POS_COUNT=$(echo "$POSITIONS" | grep -o '"posId"' | wc -l | tr -d ' ')

    # Get current algo orders
    ALGOS=$(curl -s "$SIM_URL/api/v5/trade/orders-algo-pending?instId=BTC-USDT-SWAP" 2>/dev/null)
    CURR_SL=$(echo "$ALGOS" | grep -o '"slTriggerPx":"[^"]*"' | head -1 | cut -d'"' -f4)
    CURR_TP=$(echo "$ALGOS" | grep -o '"tpTriggerPx":"[^"]*"' | head -1 | cut -d'"' -f4)
    CURR_ALGO_ID=$(echo "$ALGOS" | grep -o '"algoId":"[^"]*"' | head -1 | cut -d'"' -f4)

    # Record initial TPSL set
    if [ "$INITIAL_SL_RECORDED" = false ] && [ -n "$CURR_SL" ] && [ "$CURR_SL" != "0.00000000" ]; then
        echo "  ✓ [t+${ELAPSED}s] Initial SL set: $CURR_SL (TP: $CURR_TP)"
        INITIAL_SL_RECORDED=true
        LAST_SL="$CURR_SL"
        TPSL_ALGO_ID="$CURR_ALGO_ID"
    fi

    # Record SL changes
    if [ -n "$CURR_SL" ] && [ "$CURR_SL" != "0.00000000" ] && [ "$CURR_SL" != "$LAST_SL" ] && [ "$INITIAL_SL_RECORDED" = true ]; then
        if [ "$PHASE_A_RECORDED" = false ]; then
            echo "  ✓ [t+${ELAPSED}s] Phase A: SL adjusted ${LAST_SL} → ${CURR_SL} (price=$CURR_PRICE)"
            PHASE_A_RECORDED=true
        else
            PHASE_B_COUNT=$((PHASE_B_COUNT + 1))
            echo "  ✓ [t+${ELAPSED}s] Phase B (#${PHASE_B_COUNT}): SL adjusted ${LAST_SL} → ${CURR_SL} (price=$CURR_PRICE)"
        fi
        LAST_SL="$CURR_SL"
    fi

    # Check if position closed (SL triggered)
    if [ "$INITIAL_SL_RECORDED" = true ] && [ "$POS_COUNT" -eq 0 ]; then
        SL_TRIGGER_PRICE="$CURR_PRICE"
        echo ""
        echo "  ✓ [t+${ELAPSED}s] SL TRIGGERED at price=$CURR_PRICE (expected SL=$LAST_SL)"
        break
    fi

    printf "  [t+${ELAPSED}s] price=$CURR_PRICE | algos=$POS_COUNT pos | SL=$CURR_SL\r"
    sleep 5
done

echo ""
echo ""
echo "=========================================="
echo "Test $TEST_NUM Results:"
echo "  Initial SL: $(grep -m1 "Initial SL" <<< "$(grep "Initial SL" "$LOG_DIR/tenyojubaku.log" 2>/dev/null)" | head -1 || echo "see above")"
echo "  Final SL before trigger: $LAST_SL"
echo "  SL trigger price: $SL_TRIGGER_PRICE"
echo "  Phase B adjustments: $PHASE_B_COUNT"
echo "=========================================="

# Save summary
cat > "$LOG_DIR/summary.txt" << EOF
Test $TEST_NUM Summary
====================
Config: volatility_pct=$VOLATILITY, check_interval=20s, monitoring.interval=5s
Side: $SIDE_LABEL
Entry: 100
Final SL: $LAST_SL
SL Triggered at: $SL_TRIGGER_PRICE
Phase B adjustments: $PHASE_B_COUNT
EOF

# Cleanup
kill $SIM_PID 2>/dev/null
kill $TJ_PID 2>/dev/null
wait 2>/dev/null

echo ""
echo "Logs saved to $LOG_DIR/"
echo "Simulator log: $LOG_DIR/simulator.log"
echo "TenyoJubaku log: $LOG_DIR/tenyojubaku.log"
