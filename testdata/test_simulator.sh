#!/bin/bash
# OKX Simulator 测试脚本

set -e

CLI="./bin/tenyojubaku-cli"
SIMULATOR="./bin/okx-simulator"

echo "========================================="
echo "  OKX Simulator 测试"
echo "========================================="
echo ""

# 测试 1: 基本下单和查询
echo "测试 1: 基本下单和查询"
echo "-------------------------------------"
echo "1. 启动 simulator (价格固定在 100)"
$SIMULATOR 100 100 100 &
SIM_PID=$!
sleep 2

echo "2. 下单 long 0.01 BTC @ 100"
$CLI order place --instrument BTC-USDT-SWAP --side buy --size 0.01 --price 100

sleep 1

echo "3. 查询仓位"
$CLI position list

sleep 1

echo "4. 停止 simulator"
kill $SIM_PID
echo ""
echo "✓ 测试 1 完成"
echo ""

sleep 2

# 测试 2: 止盈止损
echo "测试 2: 止盈止损触发"
echo "-------------------------------------"
echo "1. 启动 simulator (价格会上涨到 110)"
$SIMULATOR -file=testdata/test_tpsl.txt -interval=0.05 &
SIM_PID=$!
sleep 2

echo "2. 下单 long 0.01 BTC @ 100"
$CLI order place --instrument BTC-USDT-SWAP --side buy --size 0.01 --price 100

sleep 1

echo "3. 设置止盈 TP=110, SL=95"
$CLI tpsl set --instrument BTC-USDT-SWAP --tp 110 --sl 95

sleep 1

echo "4. 查询仓位和 TPSL 订单"
$CLI position list
$CLI tpsl list

echo "5. 等待价格上涨触发 TP..."
sleep 10

echo "6. 查询仓位（应该已平仓）"
$CLI position list

echo "7. 停止 simulator"
kill $SIM_PID
echo ""
echo "✓ 测试 2 完成"
echo ""

sleep 2

# 测试 3: 爆仓
echo "测试 3: 爆仓测试"
echo "-------------------------------------"
echo "1. 启动 simulator (价格会下跌到 89)"
$SIMULATOR -file=testdata/test_liquidation.txt -interval=0.05 &
SIM_PID=$!
sleep 2

echo "2. 下单 long 0.01 BTC @ 100 (杠杆 10x, 爆仓价 91)"
$CLI order place --instrument BTC-USDT-SWAP --side buy --size 0.01 --price 100

sleep 1

echo "3. 查询仓位（应显示爆仓价格 liqPx=91）"
$CLI position list

echo "4. 等待价格下跌触发爆仓..."
sleep 10

echo "5. 查询仓位（应该已被强平）"
$CLI position list

echo "6. 停止 simulator"
kill $SIM_PID
echo ""
echo "✓ 测试 3 完成"
echo ""

echo "========================================="
echo "  所有测试完成"
echo "========================================="
