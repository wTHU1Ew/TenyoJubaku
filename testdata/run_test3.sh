#!/bin/bash
# Test 3: 只设TP，价格触发爆仓
# entry=100, 10x leverage → liqPx = 100 * (1 - 0.1 + 0.01) = 91
# 价格序列: 100→98→95→93→91→89→87
# SL=95 但只设置TP，所以当价格跌到91时爆仓

SIM_URL="http://localhost:8888"
BIN_DIR="$(dirname $0)/../bin"
INTERVAL=0.35  # 21 seconds per price change

echo "=========================================="
echo "Test 3: 只设TP + 爆仓测试"
echo "=========================================="
echo "entry=100, lever=10x, liqPx=91"
echo "只设 TP=110，不设 SL"
echo "价格会跌至 91 触发爆仓"

echo ""
echo "[1/6] 启动 simulator..."
"$BIN_DIR/okx-simulator" -file="$(dirname $0)/test3_prices.txt" -interval=$INTERVAL > /tmp/sim_test3.log 2>&1 &
SIM_PID=$!
echo "Simulator PID: $SIM_PID"
sleep 2

# 确认价格
TICKER=$(curl -s "$SIM_URL/api/v5/market/ticker?instId=BTC-USDT-SWAP")
CURR_PRICE=$(echo $TICKER | grep -o '"last":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "当前价格: $CURR_PRICE"

# 下多单
echo ""
echo "[2/6] 下多单 0.01 BTC..."
ORDER=$(curl -s -X POST "$SIM_URL/api/v5/trade/order" \
  -H "Content-Type: application/json" \
  -d '{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"buy","ordType":"market","sz":"0.01"}')
echo "Order 结果: $ORDER"
sleep 1

# 查询仓位确认 liqPx
POS=$(curl -s "$SIM_URL/api/v5/account/positions?instId=BTC-USDT-SWAP")
LIQ_PX=$(echo $POS | grep -o '"liqPx":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "仓位 liqPx: $LIQ_PX (预期 91)"

# 只设 TP=110（不设SL），让价格跌穿爆仓线
echo ""
echo "[3/6] 只设 TP=110（不设SL）..."
ALGO=$(curl -s -X POST "$SIM_URL/api/v5/trade/order-algo" \
  -H "Content-Type: application/json" \
  -d '{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"sell","posSide":"long","ordType":"oco","sz":"0.01","tpTriggerPx":"110","tpOrdPx":"-1","tpTriggerPxType":"last"}')
echo "Algo 结果: $ALGO"

echo ""
echo "等待爆仓触发 (price <= $LIQ_PX)..."
echo "每5秒检查一次仓位..."

ATTEMPTS=0
MAX_ATTEMPTS=30
while [ $ATTEMPTS -lt $MAX_ATTEMPTS ]; do
  sleep 5
  POSITIONS=$(curl -s "$SIM_URL/api/v5/account/positions?instId=BTC-USDT-SWAP")
  POS_COUNT=$(echo $POSITIONS | grep -o '"posId"' | wc -l | tr -d ' ')

  TICKER=$(curl -s "$SIM_URL/api/v5/market/ticker?instId=BTC-USDT-SWAP")
  CURR_PRICE=$(echo $TICKER | grep -o '"last":"[^"]*"' | head -1 | cut -d'"' -f4)

  echo "  [t+$((ATTEMPTS*5+5))s] 价格=$CURR_PRICE 仓位数=$POS_COUNT"

  if [ "$POS_COUNT" -eq 0 ]; then
    echo "  ✓ 爆仓已触发！仓位已关闭"
    break
  fi

  ATTEMPTS=$((ATTEMPTS + 1))
done

if [ $ATTEMPTS -ge $MAX_ATTEMPTS ]; then
  echo "  ✗ 超时：爆仓未触发"
  cat /tmp/sim_test3.log
  kill $SIM_PID 2>/dev/null
  exit 1
fi

# 查询最终状态
echo ""
echo "[6/6] 查询最终状态..."
FINAL_POS=$(curl -s "$SIM_URL/api/v5/account/positions?instId=BTC-USDT-SWAP")
echo "最终仓位: $FINAL_POS"

PENDING_ORDERS=$(curl -s "$SIM_URL/api/v5/trade/orders-pending?instId=BTC-USDT-SWAP")
echo "待成交订单: $PENDING_ORDERS"

PENDING_ALGO=$(curl -s "$SIM_URL/api/v5/trade/orders-algo-pending?instId=BTC-USDT-SWAP")
echo "待成交Algo: $PENDING_ALGO"

echo ""
echo "=========================================="
echo "Test 3 完成 - Simulator 日志:"
echo "=========================================="
cat /tmp/sim_test3.log

kill $SIM_PID 2>/dev/null
echo "Simulator 已停止"
