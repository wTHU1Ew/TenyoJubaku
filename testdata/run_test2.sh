#!/bin/bash
# Test 2: 两次TPSL触发测试
# 价格序列: 100→102→105→108→110→110→112→115→118→120
# Order 1: entry=100, TP=110 (第5个价格触发)
# Order 2: entry=110, TP=120 (第10个价格触发)

SIM_URL="http://localhost:8888"
BIN_DIR="$(dirname $0)/../bin"
INTERVAL=0.35  # 21 seconds per price change, 足够时间在触发前设置TPSL

echo "=========================================="
echo "Test 2: 两次TPSL触发测试"
echo "=========================================="

# 启动 simulator
echo "[1/8] 启动 simulator (interval=${INTERVAL}min)..."
"$BIN_DIR/okx-simulator" -file="$(dirname $0)/test2_prices.txt" -interval=$INTERVAL > /tmp/sim_test2.log 2>&1 &
SIM_PID=$!
echo "Simulator PID: $SIM_PID"
sleep 2

# 确认 simulator 已启动
TICKER=$(curl -s "$SIM_URL/api/v5/market/ticker?instId=BTC-USDT-SWAP")
PRICE=$(echo $TICKER | python3 -c "import sys,json; d=json.load(sys.stdin); print(d['data'][0]['last'])" 2>/dev/null || echo $TICKER | grep -o '"last":"[^"]*"' | head -1)
echo "当前价格: $PRICE"

# ============= 第一笔交易 =============
echo ""
echo "[2/8] 第一笔：下多单 0.01 BTC..."
ORDER1=$(curl -s -X POST "$SIM_URL/api/v5/trade/order" \
  -H "Content-Type: application/json" \
  -d '{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"buy","ordType":"market","sz":"0.01"}')
echo "Order 1 结果: $ORDER1"
ORD_ID1=$(echo $ORDER1 | grep -o '"ordId":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Order ID: $ORD_ID1"
sleep 1

echo "[3/8] 第一笔：设置 TPSL (TP=110, SL=95)..."
ALGO1=$(curl -s -X POST "$SIM_URL/api/v5/trade/order-algo" \
  -H "Content-Type: application/json" \
  -d '{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"sell","posSide":"long","ordType":"oco","sz":"0.01","tpTriggerPx":"110","tpOrdPx":"-1","slTriggerPx":"95","slOrdPx":"-1","tpTriggerPxType":"last","slTriggerPxType":"last"}')
echo "Algo 1 结果: $ALGO1"
ALGO_ID1=$(echo $ALGO1 | grep -o '"algoId":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Algo ID: $ALGO_ID1"

echo ""
echo "等待第一次TP触发 (price=110)... 预计约 $(echo "$INTERVAL * 4 * 60" | bc)s"
echo "每5秒检查一次仓位..."

# 等待第一次TP触发
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
    echo "  ✓ 第一次TP已触发！仓位已关闭"
    break
  fi

  ATTEMPTS=$((ATTEMPTS + 1))
done

if [ $ATTEMPTS -ge $MAX_ATTEMPTS ]; then
  echo "  ✗ 超时：第一次TP未触发"
  kill $SIM_PID 2>/dev/null
  exit 1
fi

# ============= 第二笔交易 =============
echo ""
echo "[5/8] 第二笔：当前价格下继续下多单 0.01 BTC..."
ORDER2=$(curl -s -X POST "$SIM_URL/api/v5/trade/order" \
  -H "Content-Type: application/json" \
  -d '{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"buy","ordType":"market","sz":"0.01"}')
echo "Order 2 结果: $ORDER2"
ORD_ID2=$(echo $ORDER2 | grep -o '"ordId":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Order ID: $ORD_ID2"
sleep 1

echo "[6/8] 第二笔：设置 TPSL (TP=120, SL=105)..."
ALGO2=$(curl -s -X POST "$SIM_URL/api/v5/trade/order-algo" \
  -H "Content-Type: application/json" \
  -d '{"instId":"BTC-USDT-SWAP","tdMode":"cross","side":"sell","posSide":"long","ordType":"oco","sz":"0.01","tpTriggerPx":"120","tpOrdPx":"-1","slTriggerPx":"105","slOrdPx":"-1","tpTriggerPxType":"last","slTriggerPxType":"last"}')
echo "Algo 2 结果: $ALGO2"
ALGO_ID2=$(echo $ALGO2 | grep -o '"algoId":"[^"]*"' | head -1 | cut -d'"' -f4)
echo "Algo ID: $ALGO_ID2"

echo ""
echo "等待第二次TP触发 (price=120)..."

ATTEMPTS=0
while [ $ATTEMPTS -lt $MAX_ATTEMPTS ]; do
  sleep 5
  POSITIONS=$(curl -s "$SIM_URL/api/v5/account/positions?instId=BTC-USDT-SWAP")
  POS_COUNT=$(echo $POSITIONS | grep -o '"posId"' | wc -l | tr -d ' ')

  TICKER=$(curl -s "$SIM_URL/api/v5/market/ticker?instId=BTC-USDT-SWAP")
  CURR_PRICE=$(echo $TICKER | grep -o '"last":"[^"]*"' | head -1 | cut -d'"' -f4)

  echo "  [t+$((ATTEMPTS*5+5))s] 价格=$CURR_PRICE 仓位数=$POS_COUNT"

  if [ "$POS_COUNT" -eq 0 ]; then
    echo "  ✓ 第二次TP已触发！仓位已关闭"
    break
  fi

  ATTEMPTS=$((ATTEMPTS + 1))
done

if [ $ATTEMPTS -ge $MAX_ATTEMPTS ]; then
  echo "  ✗ 超时：第二次TP未触发"
  kill $SIM_PID 2>/dev/null
  exit 1
fi

# ============= 查询历史记录 =============
echo ""
echo "[8/8] 查询所有历史订单..."
PENDING=$(curl -s "$SIM_URL/api/v5/trade/orders-pending?instId=BTC-USDT-SWAP")
echo "当前待成交订单: $PENDING"

ALGO_PENDING=$(curl -s "$SIM_URL/api/v5/trade/orders-algo-pending?instId=BTC-USDT-SWAP")
echo "当前待成交algo订单: $ALGO_PENDING"

FINAL_POS=$(curl -s "$SIM_URL/api/v5/account/positions?instId=BTC-USDT-SWAP")
echo "最终仓位: $FINAL_POS"

echo ""
echo "=========================================="
echo "Test 2 完成 - Simulator 日志:"
echo "=========================================="
cat /tmp/sim_test2.log

kill $SIM_PID 2>/dev/null
echo "Simulator 已停止"
