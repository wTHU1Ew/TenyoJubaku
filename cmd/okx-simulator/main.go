// OKX Simulator - OKX交易所模拟器
// A standalone tool that simulates OKX exchange with order matching, position management, and liquidation.
//
// Usage:
//   ./okx-simulator [flags] <price1> <price2> ...
//   ./okx-simulator -file=prices.txt [flags]
//
// Flags:
//   -interval=3    Price change interval in minutes (default: 3)
//   -file=FILE     Load prices from file (one price per line, # for comments)
//   -port=8888     HTTP server port (default: 8888, localhost only)
//
// Examples:
//   ./okx-simulator 100 101 102 103 104 105
//   ./okx-simulator -file=prices.txt -interval=1

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	DEFAULT_LEVERAGE      = 10.0
	MAINTENANCE_MARGIN_RATIO = 0.01 // 1% maintenance margin for liquidation
)

// ========================================
// Core Data Structures
// ========================================

// Position 仓位 / Position
type Position struct {
	InstId     string
	PosSide    string  // "long" or "short"
	Size       float64 // Position size
	AvgPx      float64 // Average entry price
	Lever      float64 // Leverage
	Margin     float64 // Initial margin
	CTime      int64   // Creation time
	UTime      int64   // Update time
}

// Order 订单 / Order
type Order struct {
	OrdId      string
	InstId     string
	Side       string  // "buy" or "sell"
	PosSide    string  // "long" or "short" for hedge mode
	OrdType    string  // "limit", "market"
	Sz         float64 // Order size
	Px         float64 // Order price (0 for market orders)
	State      string  // "live", "filled", "canceled"
	AccFillSz  float64 // Accumulated fill size
	AvgPx      float64 // Average fill price
	CTime      int64
	UTime      int64
	ReduceOnly bool
}

// AlgoOrderData 止盈止损订单 / TPSL Order
type AlgoOrderData struct {
	AlgoId          string
	InstId          string
	PosSide         string
	Side            string
	OrdType         string  // "conditional"
	Sz              float64
	TpTriggerPx     float64
	SlTriggerPx     float64
	TpOrdPx         float64 // -1 for market
	SlOrdPx         float64 // -1 for market
	TpTriggerPxType string  // "last", "index", "mark"
	SlTriggerPxType string
	State           string  // "live", "effective", "canceled"
	CTime           int64
	TriggerTime     int64
	ReduceOnly      bool
}

// ========================================
// Trading Engine - 交易引擎
// ========================================

type TradingEngine struct {
	mu            sync.RWMutex

	// State
	positions     map[string]*Position  // key: instId_posSide
	orders        map[string]*Order     // key: ordId
	algoOrders    map[string]*AlgoOrderData  // key: algoId

	// ID generators
	orderIdCounter    uint64
	algoOrderIdCounter uint64

	// Account
	balance       float64 // Total balance in USDT

	// Price reference
	priceCtrl     *PriceController
}

func NewTradingEngine(priceCtrl *PriceController, initialBalance float64) *TradingEngine {
	return &TradingEngine{
		positions:  make(map[string]*Position),
		orders:     make(map[string]*Order),
		algoOrders: make(map[string]*AlgoOrderData),
		balance:    initialBalance,
		priceCtrl:  priceCtrl,
	}
}

// PlaceOrder 下单 / Place order
func (te *TradingEngine) PlaceOrder(req *OrderRequest) (*Order, error) {
	te.mu.Lock()
	defer te.mu.Unlock()

	ordId := fmt.Sprintf("ord-%d", atomic.AddUint64(&te.orderIdCounter, 1))
	sz, _ := strconv.ParseFloat(req.Sz, 64)
	px, _ := strconv.ParseFloat(req.Px, 64)

	// Auto-infer posSide if not provided
	posSide := req.PosSide
	if posSide == "" {
		if req.Side == "buy" {
			posSide = "long"
		} else if req.Side == "sell" {
			posSide = "short"
		}
	}

	order := &Order{
		OrdId:      ordId,
		InstId:     req.InstId,
		Side:       req.Side,
		PosSide:    posSide,
		OrdType:    req.OrdType,
		Sz:         sz,
		Px:         px,
		State:      "live",
		CTime:      time.Now().UnixMilli(),
		UTime:      time.Now().UnixMilli(),
		ReduceOnly: req.ReduceOnly,
	}

	// Market order: execute immediately
	if req.OrdType == "market" || req.OrdType == "limit" {
		currentPrice := te.priceCtrl.GetCurrentPrice()
		if req.OrdType == "market" {
			order.Px = currentPrice
		}
		// Execute immediately for simplicity
		te.executeOrder(order, currentPrice)
	}

	te.orders[ordId] = order
	return order, nil
}

// executeOrder 执行订单 / Execute order
func (te *TradingEngine) executeOrder(order *Order, price float64) {
	order.State = "filled"
	order.AccFillSz = order.Sz
	order.AvgPx = price
	order.UTime = time.Now().UnixMilli()

	// Update position
	posKey := order.InstId + "_" + order.PosSide
	pos, exists := te.positions[posKey]

	if order.ReduceOnly {
		// Close position
		if exists {
			pos.Size -= order.Sz
			if pos.Size <= 0 {
				delete(te.positions, posKey)
			}
			pos.UTime = time.Now().UnixMilli()
		}
	} else {
		// Open or add to position
		if !exists {
			// New position
			leverage := DEFAULT_LEVERAGE
			margin := (order.Sz * price) / leverage

			pos = &Position{
				InstId:  order.InstId,
				PosSide: order.PosSide,
				Size:    order.Sz,
				AvgPx:   price,
				Lever:   leverage,
				Margin:  margin,
				CTime:   time.Now().UnixMilli(),
				UTime:   time.Now().UnixMilli(),
			}
			te.positions[posKey] = pos

			log.Printf("[EXEC] New position: %s %s size=%.4f entry=%.2f lever=%.0f",
				pos.InstId, pos.PosSide, pos.Size, pos.AvgPx, pos.Lever)
		} else {
			// Add to existing position
			totalValue := pos.Size*pos.AvgPx + order.Sz*price
			pos.Size += order.Sz
			pos.AvgPx = totalValue / pos.Size
			pos.UTime = time.Now().UnixMilli()

			log.Printf("[EXEC] Updated position: %s %s size=%.4f avgPx=%.2f",
				pos.InstId, pos.PosSide, pos.Size, pos.AvgPx)
		}
	}

	log.Printf("[EXEC] Order filled: ordId=%s, price=%.2f, sz=%.4f", order.OrdId, price, order.Sz)
}

// PlaceAlgoOrder 下止盈止损单 / Place TPSL order
func (te *TradingEngine) PlaceAlgoOrder(req *AlgoOrderRequest) (*AlgoOrderData, error) {
	te.mu.Lock()
	defer te.mu.Unlock()

	algoId := fmt.Sprintf("algo-%d", atomic.AddUint64(&te.algoOrderIdCounter, 1))
	sz, _ := strconv.ParseFloat(req.Sz, 64)
	tpTrigger, _ := strconv.ParseFloat(req.TpTriggerPx, 64)
	slTrigger, _ := strconv.ParseFloat(req.SlTriggerPx, 64)
	tpOrd, _ := strconv.ParseFloat(req.TpOrdPx, 64)
	slOrd, _ := strconv.ParseFloat(req.SlOrdPx, 64)

	algo := &AlgoOrderData{
		AlgoId:          algoId,
		InstId:          req.InstId,
		PosSide:         req.PosSide,
		Side:            req.Side,
		OrdType:         req.OrdType,
		Sz:              sz,
		TpTriggerPx:     tpTrigger,
		SlTriggerPx:     slTrigger,
		TpOrdPx:         tpOrd,
		SlOrdPx:         slOrd,
		TpTriggerPxType: req.TpTriggerPxType,
		SlTriggerPxType: req.SlTriggerPxType,
		State:           "live",
		CTime:           time.Now().UnixMilli(),
		ReduceOnly:      req.ReduceOnly,
	}

	te.algoOrders[algoId] = algo

	log.Printf("[ALGO] Placed TPSL: algoId=%s, tp=%.2f, sl=%.2f", algoId, tpTrigger, slTrigger)
	return algo, nil
}

// AmendAlgoOrder 修改止盈止损 / Amend TPSL order
func (te *TradingEngine) AmendAlgoOrder(req *AmendAlgoRequest) (*AlgoOrderData, error) {
	te.mu.Lock()
	defer te.mu.Unlock()

	algo, exists := te.algoOrders[req.AlgoId]
	if !exists {
		return nil, fmt.Errorf("algo order not found: %s", req.AlgoId)
	}

	if req.NewTpTriggerPx != "" {
		algo.TpTriggerPx, _ = strconv.ParseFloat(req.NewTpTriggerPx, 64)
	}
	if req.NewSlTriggerPx != "" {
		algo.SlTriggerPx, _ = strconv.ParseFloat(req.NewSlTriggerPx, 64)
	}
	if req.NewTpOrdPx != "" {
		algo.TpOrdPx, _ = strconv.ParseFloat(req.NewTpOrdPx, 64)
	}
	if req.NewSlOrdPx != "" {
		algo.SlOrdPx, _ = strconv.ParseFloat(req.NewSlOrdPx, 64)
	}

	log.Printf("[ALGO] Amended TPSL: algoId=%s, newTp=%.2f, newSl=%.2f",
		req.AlgoId, algo.TpTriggerPx, algo.SlTriggerPx)

	return algo, nil
}

// OnPriceChange 价格变化时的处理 / Handle price change
func (te *TradingEngine) OnPriceChange(newPrice float64) {
	te.mu.Lock()
	defer te.mu.Unlock()

	// 1. Check limit orders for execution
	te.checkLimitOrders(newPrice)

	// 2. Check TPSL triggers
	te.checkAlgoOrders(newPrice)

	// 3. Check liquidation
	te.checkLiquidation(newPrice)
}

// checkLimitOrders 检查限价单是否可成交 / Check limit orders
func (te *TradingEngine) checkLimitOrders(price float64) {
	for _, order := range te.orders {
		if order.State != "live" || order.OrdType != "limit" {
			continue
		}

		shouldFill := false
		if order.Side == "buy" && price <= order.Px {
			shouldFill = true
		} else if order.Side == "sell" && price >= order.Px {
			shouldFill = true
		}

		if shouldFill {
			te.executeOrder(order, price)
		}
	}
}

// checkAlgoOrders 检查止盈止损触发 / Check TPSL triggers
func (te *TradingEngine) checkAlgoOrders(price float64) {
	for _, algo := range te.algoOrders {
		if algo.State != "live" {
			continue
		}

		// Direction-aware trigger logic / 方向感知触发逻辑
		// Long: TP triggers when price rises (>=), SL triggers when price drops (<=)
		// Short: TP triggers when price drops (<=), SL triggers when price rises (>=)
		isShortAlgo := algo.PosSide == "short"
		var tpTriggered, slTriggered bool
		if isShortAlgo {
			tpTriggered = algo.TpTriggerPx > 0 && price <= algo.TpTriggerPx
			slTriggered = algo.SlTriggerPx > 0 && price >= algo.SlTriggerPx
		} else {
			tpTriggered = algo.TpTriggerPx > 0 && price >= algo.TpTriggerPx
			slTriggered = algo.SlTriggerPx > 0 && price <= algo.SlTriggerPx
		}

		if tpTriggered || slTriggered {
			triggerPrice := price
			if tpTriggered {
				if algo.TpOrdPx > 0 {
					triggerPrice = algo.TpOrdPx
				}
				log.Printf("[TPSL] Take-profit triggered: algoId=%s, price=%.2f", algo.AlgoId, price)
			} else {
				if algo.SlOrdPx > 0 {
					triggerPrice = algo.SlOrdPx
				}
				log.Printf("[TPSL] Stop-loss triggered: algoId=%s, price=%.2f", algo.AlgoId, price)
			}

			// Execute close order
			order := &Order{
				OrdId:      "tpsl-" + algo.AlgoId,
				InstId:     algo.InstId,
				Side:       algo.Side,
				PosSide:    algo.PosSide,
				OrdType:    "market",
				Sz:         algo.Sz,
				Px:         triggerPrice,
				State:      "live",
				CTime:      time.Now().UnixMilli(),
				UTime:      time.Now().UnixMilli(),
				ReduceOnly: true,
			}

			te.executeOrder(order, triggerPrice)

			// Mark algo order as effective
			algo.State = "effective"
			algo.TriggerTime = time.Now().UnixMilli()
		}
	}
}

// checkLiquidation 检查爆仓 / Check liquidation
func (te *TradingEngine) checkLiquidation(price float64) {
	for posKey, pos := range te.positions {
		liqPrice := te.calculateLiquidationPrice(pos)

		shouldLiquidate := false
		if pos.PosSide == "long" && price <= liqPrice {
			shouldLiquidate = true
		} else if pos.PosSide == "short" && price >= liqPrice {
			shouldLiquidate = true
		}

		if shouldLiquidate {
			log.Printf("[LIQUIDATION] Position liquidated: %s %s at price=%.2f (liqPx=%.2f)",
				pos.InstId, pos.PosSide, price, liqPrice)

			// Force close position
			order := &Order{
				OrdId:      "liq-" + posKey,
				InstId:     pos.InstId,
				Side:       te.getCloseSide(pos.PosSide),
				PosSide:    pos.PosSide,
				OrdType:    "market",
				Sz:         pos.Size,
				Px:         price,
				State:      "live",
				CTime:      time.Now().UnixMilli(),
				UTime:      time.Now().UnixMilli(),
				ReduceOnly: true,
			}

			te.executeOrder(order, price)
		}
	}
}

// calculateLiquidationPrice 计算爆仓价格 / Calculate liquidation price
func (te *TradingEngine) calculateLiquidationPrice(pos *Position) float64 {
	// Liquidation price formula:
	// For long: liqPx = avgPx * (1 - 1/leverage + mmr)
	// For short: liqPx = avgPx * (1 + 1/leverage - mmr)

	mmr := MAINTENANCE_MARGIN_RATIO
	leverageFactor := 1.0 / pos.Lever

	var liqPx float64
	if pos.PosSide == "long" {
		liqPx = pos.AvgPx * (1 - leverageFactor + mmr)
	} else {
		liqPx = pos.AvgPx * (1 + leverageFactor - mmr)
	}

	return liqPx
}

// getCloseSide 获取平仓方向 / Get close side
func (te *TradingEngine) getCloseSide(posSide string) string {
	if posSide == "long" {
		return "sell"
	}
	return "buy"
}

// calculateUpl 计算未实现盈亏 / Calculate unrealized PnL
func (te *TradingEngine) calculateUpl(pos *Position, currentPrice float64) float64 {
	if pos.PosSide == "long" {
		return (currentPrice - pos.AvgPx) * pos.Size
	}
	return (pos.AvgPx - currentPrice) * pos.Size
}

// GetPositions 获取所有仓位 / Get all positions
func (te *TradingEngine) GetPositions() []*Position {
	te.mu.RLock()
	defer te.mu.RUnlock()

	result := make([]*Position, 0, len(te.positions))
	for _, pos := range te.positions {
		result = append(result, pos)
	}
	return result
}

// GetOrders 获取订单列表 / Get orders
func (te *TradingEngine) GetOrders(state string) []*Order {
	te.mu.RLock()
	defer te.mu.RUnlock()

	result := make([]*Order, 0)
	for _, order := range te.orders {
		if state == "" || order.State == state {
			result = append(result, order)
		}
	}
	return result
}

// GetAlgoOrders 获取算法订单 / Get algo orders
func (te *TradingEngine) GetAlgoOrders(state string) []*AlgoOrderData {
	te.mu.RLock()
	defer te.mu.RUnlock()

	result := make([]*AlgoOrderData, 0)
	for _, algo := range te.algoOrders {
		if state == "" || algo.State == state {
			result = append(result, algo)
		}
	}
	return result
}

// ========================================
// Price Controller
// ========================================

type PriceController struct {
	mu           sync.RWMutex
	prices       []float64
	currentIndex int
	interval     time.Duration
	stopCh       chan struct{}
	engine       *TradingEngine
}

func NewPriceController(prices []float64, intervalMinutes float64) *PriceController {
	return &PriceController{
		prices:       prices,
		currentIndex: 0,
		interval:     time.Duration(intervalMinutes * float64(time.Minute)),
		stopCh:       make(chan struct{}),
	}
}

func (pc *PriceController) SetEngine(engine *TradingEngine) {
	pc.engine = engine
}

func (pc *PriceController) Start() {
	go func() {
		ticker := time.NewTicker(pc.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				pc.advance()
			case <-pc.stopCh:
				return
			}
		}
	}()
}

func (pc *PriceController) Stop() {
	close(pc.stopCh)
}

func (pc *PriceController) advance() {
	pc.mu.Lock()

	if pc.currentIndex < len(pc.prices)-1 {
		oldPrice := pc.prices[pc.currentIndex]
		pc.currentIndex++
		newPrice := pc.prices[pc.currentIndex]

		pc.mu.Unlock()

		log.Printf("[PRICE] Changed: %.2f -> %.2f (index %d/%d)",
			oldPrice, newPrice, pc.currentIndex+1, len(pc.prices))

		// Notify engine of price change
		if pc.engine != nil {
			pc.engine.OnPriceChange(newPrice)
		}
	} else {
		pc.mu.Unlock()
	}
}

func (pc *PriceController) GetCurrentPrice() float64 {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.prices[pc.currentIndex]
}

func (pc *PriceController) GetPriceString() string {
	return fmt.Sprintf("%.8f", pc.GetCurrentPrice())
}

// ========================================
// HTTP Request/Response Types
// ========================================

type BaseResponse struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

type OrderRequest struct {
	InstId     string `json:"instId"`
	TdMode     string `json:"tdMode"`
	Side       string `json:"side"`
	OrdType    string `json:"ordType"`
	Sz         string `json:"sz"`
	Px         string `json:"px"`
	PosSide    string `json:"posSide"`
	ReduceOnly bool   `json:"reduceOnly"`
}

type AlgoOrderRequest struct {
	InstId          string `json:"instId"`
	TdMode          string `json:"tdMode"`
	Side            string `json:"side"`
	PosSide         string `json:"posSide"`
	OrdType         string `json:"ordType"`
	Sz              string `json:"sz"`
	TpTriggerPx     string `json:"tpTriggerPx"`
	TpOrdPx         string `json:"tpOrdPx"`
	SlTriggerPx     string `json:"slTriggerPx"`
	SlOrdPx         string `json:"slOrdPx"`
	TpTriggerPxType string `json:"tpTriggerPxType"`
	SlTriggerPxType string `json:"slTriggerPxType"`
	ReduceOnly      bool   `json:"reduceOnly"`
}

type AmendAlgoRequest struct {
	AlgoId             string `json:"algoId"`
	InstId             string `json:"instId"`
	NewSz              string `json:"newSz"`
	NewTpTriggerPx     string `json:"newTpTriggerPx"`
	NewTpOrdPx         string `json:"newTpOrdPx"`
	NewSlTriggerPx     string `json:"newSlTriggerPx"`
	NewSlOrdPx         string `json:"newSlOrdPx"`
	NewTpTriggerPxType string `json:"newTpTriggerPxType"`
	NewSlTriggerPxType string `json:"newSlTriggerPxType"`
}

// ========================================
// HTTP Server
// ========================================

type Server struct {
	priceCtrl *PriceController
	engine    *TradingEngine
}

func NewServer(priceCtrl *PriceController, engine *TradingEngine) *Server {
	return &Server{
		priceCtrl: priceCtrl,
		engine:    engine,
	}
}

func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// handleTicker GET /api/v5/market/ticker
func (s *Server) handleTicker(w http.ResponseWriter, r *http.Request) {
	instId := r.URL.Query().Get("instId")
	if instId == "" {
		instId = "BTC-USDT-SWAP"
	}

	price := s.priceCtrl.GetPriceString()
	currentPrice := s.priceCtrl.GetCurrentPrice()

	resp := BaseResponse{
		Code: "0",
		Msg:  "",
		Data: []map[string]string{
			{
				"instId":  instId,
				"last":    price,
				"bidPx":   fmt.Sprintf("%.8f", currentPrice*0.9999),
				"askPx":   fmt.Sprintf("%.8f", currentPrice*1.0001),
				"lastSz":  "1",
				"open24h": price,
				"high24h": price,
				"low24h":  price,
				"ts":      fmt.Sprintf("%d", time.Now().UnixMilli()),
			},
		},
	}

	jsonResponse(w, resp)
}

// handlePositions GET /api/v5/account/positions
func (s *Server) handlePositions(w http.ResponseWriter, r *http.Request) {
	positions := s.engine.GetPositions()
	currentPrice := s.priceCtrl.GetCurrentPrice()

	data := make([]map[string]string, 0, len(positions))
	for _, pos := range positions {
		upl := s.engine.calculateUpl(pos, currentPrice)
		uplRatio := upl / (pos.AvgPx * pos.Size) * pos.Lever
		liqPx := s.engine.calculateLiquidationPrice(pos)

		data = append(data, map[string]string{
			"instType": "SWAP",
			"instId":   pos.InstId,
			"mgnMode":  "cross",
			"posId":    fmt.Sprintf("pos-%s-%s", pos.InstId, pos.PosSide),
			"posSide":  pos.PosSide,
			"pos":      fmt.Sprintf("%.8f", pos.Size),
			"avgPx":    fmt.Sprintf("%.8f", pos.AvgPx),
			"upl":      fmt.Sprintf("%.8f", upl),
			"uplRatio": fmt.Sprintf("%.8f", uplRatio),
			"lever":    fmt.Sprintf("%.0f", pos.Lever),
			"liqPx":    fmt.Sprintf("%.8f", liqPx),
			"markPx":   fmt.Sprintf("%.8f", currentPrice),
			"last":     fmt.Sprintf("%.8f", currentPrice),
			"cTime":    fmt.Sprintf("%d", pos.CTime),
			"uTime":    fmt.Sprintf("%d", pos.UTime),
		})
	}

	resp := BaseResponse{
		Code: "0",
		Msg:  "",
		Data: data,
	}

	log.Printf("[GET] /api/v5/account/positions -> %d positions", len(data))
	jsonResponse(w, resp)
}

// handleAccountBalance GET /api/v5/account/balance
func (s *Server) handleAccountBalance(w http.ResponseWriter, r *http.Request) {
	resp := BaseResponse{
		Code: "0",
		Msg:  "",
		Data: []map[string]interface{}{
			{
				"totalEq": fmt.Sprintf("%.2f", s.engine.balance),
				"details": []map[string]string{
					{
						"ccy":      "USDT",
						"eq":       fmt.Sprintf("%.2f", s.engine.balance),
						"availBal": fmt.Sprintf("%.2f", s.engine.balance),
					},
				},
			},
		},
	}

	jsonResponse(w, resp)
}

// handlePlaceOrder POST /api/v5/trade/order
func (s *Server) handlePlaceOrder(w http.ResponseWriter, r *http.Request) {
	var req OrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := BaseResponse{Code: "50000", Msg: "Invalid request body", Data: []interface{}{}}
		jsonResponse(w, resp)
		return
	}

	order, err := s.engine.PlaceOrder(&req)
	if err != nil {
		resp := BaseResponse{Code: "50001", Msg: err.Error(), Data: []interface{}{}}
		jsonResponse(w, resp)
		return
	}

	resp := BaseResponse{
		Code: "0",
		Msg:  "",
		Data: []map[string]string{
			{
				"ordId":   order.OrdId,
				"clOrdId": "",
				"tag":     "",
				"sCode":   "0",
				"sMsg":    "",
			},
		},
	}

	log.Printf("[POST] /api/v5/trade/order -> ordId=%s, instId=%s, side=%s, sz=%s",
		order.OrdId, req.InstId, req.Side, req.Sz)
	jsonResponse(w, resp)
}

// handlePendingOrders GET /api/v5/trade/orders-pending
func (s *Server) handlePendingOrders(w http.ResponseWriter, r *http.Request) {
	orders := s.engine.GetOrders("live")

	data := make([]map[string]string, 0, len(orders))
	for _, order := range orders {
		data = append(data, map[string]string{
			"ordId":     order.OrdId,
			"instId":    order.InstId,
			"side":      order.Side,
			"posSide":   order.PosSide,
			"ordType":   order.OrdType,
			"sz":        fmt.Sprintf("%.8f", order.Sz),
			"px":        fmt.Sprintf("%.8f", order.Px),
			"state":     order.State,
			"accFillSz": fmt.Sprintf("%.8f", order.AccFillSz),
			"avgPx":     fmt.Sprintf("%.8f", order.AvgPx),
			"cTime":     fmt.Sprintf("%d", order.CTime),
			"uTime":     fmt.Sprintf("%d", order.UTime),
		})
	}

	resp := BaseResponse{
		Code: "0",
		Msg:  "",
		Data: data,
	}

	log.Printf("[GET] /api/v5/trade/orders-pending -> %d orders", len(data))
	jsonResponse(w, resp)
}

// handlePlaceAlgoOrder POST /api/v5/trade/order-algo
func (s *Server) handlePlaceAlgoOrder(w http.ResponseWriter, r *http.Request) {
	var req AlgoOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := BaseResponse{Code: "50000", Msg: "Invalid request body", Data: []interface{}{}}
		jsonResponse(w, resp)
		return
	}

	algo, err := s.engine.PlaceAlgoOrder(&req)
	if err != nil {
		resp := BaseResponse{Code: "50001", Msg: err.Error(), Data: []interface{}{}}
		jsonResponse(w, resp)
		return
	}

	resp := BaseResponse{
		Code: "0",
		Msg:  "",
		Data: []map[string]string{
			{
				"algoId": algo.AlgoId,
				"sCode":  "0",
				"sMsg":   "",
			},
		},
	}

	jsonResponse(w, resp)
}

// handlePendingAlgoOrders GET /api/v5/trade/orders-algo-pending
func (s *Server) handlePendingAlgoOrders(w http.ResponseWriter, r *http.Request) {
	ordType := r.URL.Query().Get("ordType")
	algos := s.engine.GetAlgoOrders("live")

	data := make([]map[string]string, 0, len(algos))
	for _, algo := range algos {
		if ordType != "" && algo.OrdType != ordType {
			continue
		}

		data = append(data, map[string]string{
			"algoId":          algo.AlgoId,
			"instId":          algo.InstId,
			"posSide":         algo.PosSide,
			"side":            algo.Side,
			"ordType":         algo.OrdType,
			"sz":              fmt.Sprintf("%.8f", algo.Sz),
			"tpTriggerPx":     fmt.Sprintf("%.8f", algo.TpTriggerPx),
			"slTriggerPx":     fmt.Sprintf("%.8f", algo.SlTriggerPx),
			"tpOrdPx":         fmt.Sprintf("%.8f", algo.TpOrdPx),
			"slOrdPx":         fmt.Sprintf("%.8f", algo.SlOrdPx),
			"tpTriggerPxType": algo.TpTriggerPxType,
			"slTriggerPxType": algo.SlTriggerPxType,
			"state":           algo.State,
			"cTime":           fmt.Sprintf("%d", algo.CTime),
		})
	}

	resp := BaseResponse{
		Code: "0",
		Msg:  "",
		Data: data,
	}

	log.Printf("[GET] /api/v5/trade/orders-algo-pending -> %d orders", len(data))
	jsonResponse(w, resp)
}

// handleAmendAlgoOrder POST /api/v5/trade/amend-algos
func (s *Server) handleAmendAlgoOrder(w http.ResponseWriter, r *http.Request) {
	var req AmendAlgoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		resp := BaseResponse{Code: "50000", Msg: "Invalid request body", Data: []interface{}{}}
		jsonResponse(w, resp)
		return
	}

	algo, err := s.engine.AmendAlgoOrder(&req)
	if err != nil {
		resp := BaseResponse{
			Code: "51000",
			Msg:  err.Error(),
			Data: []map[string]string{
				{
					"algoId": req.AlgoId,
					"sCode":  "51000",
					"sMsg":   err.Error(),
				},
			},
		}
		jsonResponse(w, resp)
		return
	}

	resp := BaseResponse{
		Code: "0",
		Msg:  "",
		Data: []map[string]string{
			{
				"algoId": algo.AlgoId,
				"sCode":  "0",
				"sMsg":   "",
			},
		},
	}

	jsonResponse(w, resp)
}

// ========================================
// CLI Parsing
// ========================================

func loadPricesFromFile(filename string) ([]float64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var prices []float64
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		price, err := strconv.ParseFloat(line, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid price on line %d: %s", lineNum, line)
		}
		prices = append(prices, price)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return prices, nil
}

// ========================================
// Main
// ========================================

func main() {
	interval := flag.Float64("interval", 3.0, "Price change interval in minutes")
	priceFile := flag.String("file", "", "Load prices from file (one price per line)")
	port := flag.Int("port", 8888, "HTTP server port (localhost only)")
	balance := flag.Float64("balance", 100000.0, "Initial balance in USDT")

	flag.Parse()

	// Load prices
	var prices []float64
	var err error

	if *priceFile != "" {
		prices, err = loadPricesFromFile(*priceFile)
		if err != nil {
			log.Fatalf("Error loading prices from file: %v", err)
		}
	} else {
		args := flag.Args()
		if len(args) == 0 {
			fmt.Println("OKX Simulator - OKX交易所模拟器")
			fmt.Println()
			fmt.Println("Usage:")
			fmt.Println("  okx-simulator [flags] <price1> <price2> ...")
			fmt.Println("  okx-simulator -file=prices.txt [flags]")
			fmt.Println()
			fmt.Println("Flags:")
			flag.PrintDefaults()
			fmt.Println()
			fmt.Println("Examples:")
			fmt.Println("  okx-simulator 100 101 102 103 104 105")
			fmt.Println("  okx-simulator -file=prices.txt -interval=1")
			os.Exit(1)
		}

		prices = make([]float64, len(args))
		for i, arg := range args {
			prices[i], err = strconv.ParseFloat(arg, 64)
			if err != nil {
				log.Fatalf("Invalid price argument: %s", arg)
			}
		}
	}

	if len(prices) == 0 {
		log.Fatal("No prices provided")
	}

	// Validate prices
	for i, p := range prices {
		if p <= 0 || math.IsNaN(p) || math.IsInf(p, 0) {
			log.Fatalf("Invalid price at index %d: %f", i, p)
		}
	}

	// Create components
	priceCtrl := NewPriceController(prices, *interval)
	engine := NewTradingEngine(priceCtrl, *balance)
	priceCtrl.SetEngine(engine)

	// Create server
	server := NewServer(priceCtrl, engine)

	// Setup routes
	http.HandleFunc("/api/v5/market/ticker", server.handleTicker)
	http.HandleFunc("/api/v5/account/positions", server.handlePositions)
	http.HandleFunc("/api/v5/account/balance", server.handleAccountBalance)
	http.HandleFunc("/api/v5/trade/order", server.handlePlaceOrder)
	http.HandleFunc("/api/v5/trade/orders-pending", server.handlePendingOrders)
	http.HandleFunc("/api/v5/trade/order-algo", server.handlePlaceAlgoOrder)
	http.HandleFunc("/api/v5/trade/orders-algo-pending", server.handlePendingAlgoOrders)
	http.HandleFunc("/api/v5/trade/amend-algos", server.handleAmendAlgoOrder)

	// Start price controller
	priceCtrl.Start()

	// Start server
	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	log.Printf("========================================")
	log.Printf("  OKX Simulator Started")
	log.Printf("========================================")
	log.Printf("Listening on: http://%s", addr)
	log.Printf("Initial balance: %.2f USDT", *balance)
	log.Printf("Prices: %v", prices)
	log.Printf("Interval: %.1f minutes", *interval)
	log.Printf("========================================")
	log.Printf("Press Ctrl+C to stop")
	log.Printf("")

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
