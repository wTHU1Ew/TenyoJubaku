# OKX Simulator 测试指南

## 编译

```bash
go build -o bin/okx-simulator ./cmd/okx-simulator
go build -o bin/tenyojubaku-cli ./cmd/cli
```

## 测试场景

### 测试 1: 基本下单和查询

**目标**: 验证 CLI 下单后，simulator 创建仓位，TenyoJubaku 能查询到

**步骤**:

1. 启动 simulator (价格固定在 100):
```bash
./bin/okx-simulator 100 100 100
```

2. 在另一个终端，确认 config.yaml 指向 simulator:
```yaml
okx:
  api_url: "http://localhost:8888"
```

3. 下单 long 0.01 BTC:
```bash
./bin/tenyojubaku-cli order place --instrument BTC-USDT-SWAP --side buy --size 0.01 --price 100
```

4. 查询仓位:
```bash
./bin/tenyojubaku-cli position list
```

**预期结果**:
- Simulator 日志显示: `[EXEC] New position: BTC-USDT-SWAP long size=0.0100 entry=100.00 lever=10`
- CLI 查询显示仓位: `BTC-USDT-SWAP long 0.01 @ 100`

---

### 测试 2: 止盈止损触发

**目标**: 下单后设置 TPSL，价格触发止盈，仓位自动平仓

**步骤**:

1. 启动 simulator (价格会上涨到 110，每 3 秒变化):
```bash
./bin/okx-simulator -file=testdata/test_tpsl.txt -interval=0.05
```

文件内容:
```
100
101
102
105
108
110
112
```

2. 下单 long 0.01 BTC @ 100:
```bash
./bin/tenyojubaku-cli order place --instrument BTC-USDT-SWAP --side buy --size 0.01 --price 100
```

3. 查询仓位（记录 position ID）:
```bash
./bin/tenyojubaku-cli position list
```

4. 使用 TenyoJubaku 主程序或直接调用 OKX API 设置 TPSL:
```bash
# 方法1: 如果 CLI 支持
./bin/tenyojubaku-cli tpsl set --instrument BTC-USDT-SWAP --tp 110 --sl 95

# 方法2: 使用 curl 直接调用 simulator API
curl -X POST http://localhost:8888/api/v5/trade/order-algo \
  -H "Content-Type: application/json" \
  -d '{
    "instId": "BTC-USDT-SWAP",
    "tdMode": "cross",
    "side": "sell",
    "posSide": "long",
    "ordType": "conditional",
    "sz": "0.01",
    "tpTriggerPx": "110",
    "slTriggerPx": "95",
    "tpOrdPx": "-1",
    "slOrdPx": "-1"
  }'
```

5. 等待价格上涨（约 15-18 秒后价格到 110）

6. 查询仓位（应该已平仓）:
```bash
./bin/tenyojubaku-cli position list
```

**预期结果**:
- Simulator 日志显示:
  - `[PRICE] Changed: 108.00 -> 110.00`
  - `[TPSL] Take-profit triggered: algoId=algo-1, price=110.00`
  - `[EXEC] Order filled: ordId=tpsl-algo-1, price=110.00, sz=0.0100`
- CLI 查询显示无仓位

---

### 测试 3: 爆仓测试

**目标**: 下单后价格暴跌，触发爆仓，仓位强制平仓

**原理**:
- 杠杆: 10x
- 维持保证金率: 1%
- 开仓价: 100
- 爆仓价计算: liqPx = 100 * (1 - 0.1 + 0.01) = **91**

**步骤**:

1. 启动 simulator (价格会下跌到 89):
```bash
./bin/okx-simulator -file=testdata/test_liquidation.txt -interval=0.05
```

文件内容:
```
100
98
95
93
91
89
```

2. 下单 long 0.01 BTC @ 100:
```bash
./bin/tenyojubaku-cli order place --instrument BTC-USDT-SWAP --side buy --size 0.01 --price 100
```

3. 查询仓位（应显示 liqPx=91.00）:
```bash
./bin/tenyojubaku-cli position list
```

4. 等待价格下跌（约 15 秒后价格到 91）

5. 查询仓位（应该已被强平）:
```bash
./bin/tenyojubaku-cli position list
```

**预期结果**:
- Simulator 日志显示:
  - `[PRICE] Changed: 93.00 -> 91.00`
  - `[LIQUIDATION] Position liquidated: BTC-USDT-SWAP long at price=91.00 (liqPx=91.00)`
  - `[EXEC] Order filled: ordId=liq-BTC-USDT-SWAP_long, price=91.00, sz=0.0100`
- CLI 查询显示无仓位

---

## 清算价格计算公式

**做多 (long)**:
```
liqPx = avgPx * (1 - 1/leverage + mmr)
      = 100 * (1 - 0.1 + 0.01)
      = 100 * 0.91
      = 91
```

**做空 (short)**:
```
liqPx = avgPx * (1 + 1/leverage - mmr)
      = 100 * (1 + 0.1 - 0.01)
      = 100 * 1.09
      = 109
```

## 日志说明

- `[PRICE]` - 价格变化
- `[EXEC]` - 订单成交
- `[ALGO]` - 止盈止损订单操作
- `[TPSL]` - 止盈止损触发
- `[LIQUIDATION]` - 爆仓
- `[GET/POST]` - HTTP API 调用

## 故障排查

1. **下单失败 404**: 确认 config.yaml 中 `okx.api_url` 指向 `http://localhost:8888`
2. **仓位不显示**: 检查 simulator 日志是否有 `[EXEC] New position` 记录
3. **TPSL 不触发**: 确认价格已变化到触发价，检查 interval 设置
4. **爆仓不触发**: 确认价格已到达/超过清算价格
