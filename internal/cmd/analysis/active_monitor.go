package analysis

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gctx"
)

// ActiveMonitor USDT交易活跃度监控
type ActiveMonitor struct {
	tronAPI           *TronAPI
	usdtContract      string
	ctx               context.Context
	concurrency       int     // 并发数
	lookbackDays      int     // 交易回溯天数
	maxRecursionDepth int     // 最大递归深度
	prefixMaskLen     int     // 地址前缀掩码长度
	suffixMaskLen     int     // 地址后缀掩码长度
	batchSize         int     // 批处理大小
	debugMode         bool    // 调试模式
	minTxAmount       float64 // 最小交易金额
	maxTxAmount       float64 // 最大交易金额
}

// 创建活跃度监控器
func NewActiveMonitor(ctx context.Context, tronAPI *TronAPI, usdtContract string) *ActiveMonitor {
	// 从配置文件读取所有参数
	concurrency := g.Cfg().MustGet(ctx, "tron.active.concurrency", 200).Int()
	lookbackDays := g.Cfg().MustGet(ctx, "tron.active.lookbackDays", 30).Int()
	maxRecursionDepth := g.Cfg().MustGet(ctx, "tron.active.maxRecursionDepth", 100).Int()
	prefixMaskLen := g.Cfg().MustGet(ctx, "tron.active.prefixMaskLen", 3).Int()
	suffixMaskLen := g.Cfg().MustGet(ctx, "tron.active.suffixMaskLen", 4).Int()
	batchSize := g.Cfg().MustGet(ctx, "tron.active.batchSize", 100).Int()
	debugMode := g.Cfg().MustGet(ctx, "tron.active.debugMode", false).Bool()
	minTxAmount := g.Cfg().MustGet(ctx, "tron.active.minTxAmount", 1000.0).Float64()
	maxTxAmount := g.Cfg().MustGet(ctx, "tron.active.maxTxAmount", 5000.0).Float64()

	monitor := &ActiveMonitor{
		tronAPI:           tronAPI,
		usdtContract:      usdtContract,
		ctx:               gctx.New(),
		concurrency:       concurrency,       // 并发数
		lookbackDays:      lookbackDays,      // 交易回溯天数
		maxRecursionDepth: maxRecursionDepth, // 最大递归深度
		prefixMaskLen:     prefixMaskLen,     // 地址前缀掩码长度
		suffixMaskLen:     suffixMaskLen,     // 地址后缀掩码长度
		batchSize:         batchSize,         // 批处理大小
		debugMode:         debugMode,         // 调试模式
		minTxAmount:       minTxAmount,       // 最小交易金额
		maxTxAmount:       maxTxAmount,       // 最大交易金额
	}

	// 记录初始配置信息
	g.Log().Infof(monitor.ctx, "活跃度监控器初始化: 并发数=%d", monitor.concurrency)
	g.Log().Infof(monitor.ctx, "活跃度监控器初始化: 回溯天数=%d", monitor.lookbackDays)
	g.Log().Infof(monitor.ctx, "活跃度监控器初始化: 最大递归深度=%d", monitor.maxRecursionDepth)
	g.Log().Infof(monitor.ctx, "活跃度监控器初始化: 地址掩码长度=前%d后%d", monitor.prefixMaskLen, monitor.suffixMaskLen)
	g.Log().Infof(monitor.ctx, "活跃度监控器初始化: 批处理大小=%d", monitor.batchSize)
	g.Log().Infof(monitor.ctx, "活跃度监控器初始化: 调试模式=%v", monitor.debugMode)
	g.Log().Infof(monitor.ctx, "活跃度监控器初始化: USDT合约地址=%s", monitor.usdtContract)
	g.Log().Infof(monitor.ctx, "活跃度监控器初始化: 交易金额范围=%.2f-%.2f", monitor.minTxAmount, monitor.maxTxAmount)

	return monitor
}

// 启动定时监控
func (m *ActiveMonitor) StartMonitor(pattern string) error {
	_, err := gcron.Add(m.ctx, pattern, func(ctx context.Context) {
		m.AnalyzeRecentTransactions(ctx)
	}, "AnalyzeUSDTTransactions")

	if err != nil {
		return fmt.Errorf("启动USDT交易监控定时任务失败: %v", err)
	}

	g.Log().Infof(m.ctx, "USDT交易监控定时任务已启动，执行周期: %s", pattern)
	return nil
}

// 分析最近交易
func (m *ActiveMonitor) AnalyzeRecentTransactions(ctx context.Context) {
	g.Log().Info(ctx, "开始分析最近USDT交易...")

	// 1. 获取最新区块的USDT交易
	transactions, err := m.getRecentBlockUSDTTransactions(ctx)
	if err != nil {
		g.Log().Errorf(ctx, "获取最新区块USDT交易失败: %v", err)
		return
	}

	g.Log().Infof(ctx, "获取到 %d 笔符合条件的USDT交易", len(transactions))

	// 2. 提取符合初始条件的地址
	addresses := m.extractAddressesFromTransactions(ctx, transactions)
	g.Log().Infof(ctx, "提取到 %d 个初始分析地址", len(addresses))

	// 3. 递归分析活跃地址
	m.analyzeActiveAddressesRecursively(ctx, addresses, 0)
}

// 获取最新区块的USDT交易
func (m *ActiveMonitor) getRecentBlockUSDTTransactions(ctx context.Context) ([]ActiveTransaction, error) {
	// 获取最新区块
	blockResponse, err := m.tronAPI.GetLatestBlock(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取最新区块失败: %v", err)
	}

	// 从BlockResponse中提取区块号
	blockNumber := blockResponse.BlockHeader.RawData.Number
	g.Log().Infof(ctx, "获取到最新区块: %d, 包含 %d 笔交易", blockNumber, len(blockResponse.Transactions))

	// 解析区块中的交易
	transactions, err := m.tronAPI.ParseBlockTransactions(ctx, blockResponse)
	if err != nil {
		return nil, fmt.Errorf("解析区块交易失败: %v", err)
	}

	// 过滤USDT交易并转换为ActiveTransaction
	var activeTransactions []ActiveTransaction
	for _, tx := range transactions {
		// 过滤非USDT交易
		if tx.ContractAddress != m.usdtContract && tx.TokenSymbol != "USDT" {
			continue
		}

		// 过滤金额不在1000-5000之间的交易
		if tx.Amount < m.minTxAmount || tx.Amount > m.maxTxAmount {
			continue
		}

		// 转换为活跃度分析用交易格式
		activeTx := ActiveTransaction{
			TxID:           tx.TxID,
			BlockNum:       tx.BlockNumber,
			Timestamp:      tx.Timestamp,
			FromAddress:    tx.From,
			ToAddress:      tx.To,
			Amount:         tx.Amount,
			TokenType:      tx.TokenSymbol,
			ContractAddr:   tx.ContractAddress,
			TransactionFee: tx.Fee,
		}

		activeTransactions = append(activeTransactions, activeTx)
	}

	return activeTransactions, nil
}

// 获取地址的最近USDT交易
func (m *ActiveMonitor) getRecentUSDTTransactions(ctx context.Context, address string) ([]ActiveTransaction, error) {
	// 计算30天前的时间
	thirtyDaysAgo := time.Now().AddDate(0, 0, -m.lookbackDays)
	now := time.Now()

	// 使用已实现的GetUSDTTransactionsByTimeRange方法获取交易历史
	trc20Txs, err := m.tronAPI.GetUSDTTransactionsByTimeRange(ctx, address, thirtyDaysAgo, now, 1000)
	if err != nil {
		return nil, fmt.Errorf("获取地址 %s 交易历史失败: %v", address, err)
	}

	// 过滤并转换交易
	var transactions []ActiveTransaction
	for _, tx := range trc20Txs {
		// 过滤金额不在1000-5000之间的交易
		if tx.Amount < m.minTxAmount || tx.Amount > m.maxTxAmount {
			continue
		}

		// 转换为活跃度分析用交易格式
		activeTx := ActiveTransaction{
			TxID:           tx.TransactionID,
			BlockNum:       0, // TRC20Transaction中没有区块号
			Timestamp:      tx.Timestamp,
			FromAddress:    tx.From,
			ToAddress:      tx.To,
			Amount:         tx.Amount,
			TokenType:      tx.TokenSymbol,
			ContractAddr:   m.usdtContract, // 使用配置的USDT合约地址
			TransactionFee: 0,              // TRC20Transaction中没有交易费用
		}

		transactions = append(transactions, activeTx)
	}

	return transactions, nil
}

// 从交易中提取地址
func (m *ActiveMonitor) extractAddressesFromTransactions(ctx context.Context, transactions []ActiveTransaction) []string {
	addressSet := make(map[string]struct{})

	for _, tx := range transactions {
		// 添加转入和转出地址
		addressSet[tx.FromAddress] = struct{}{}
		addressSet[tx.ToAddress] = struct{}{}
	}

	// 转换为切片
	addresses := make([]string, 0, len(addressSet))
	for addr := range addressSet {
		addresses = append(addresses, addr)
	}

	return addresses
}

// 递归分析活跃地址
func (m *ActiveMonitor) analyzeActiveAddressesRecursively(ctx context.Context, addresses []string, depth int) {
	if depth >= m.maxRecursionDepth {
		g.Log().Infof(ctx, "达到最大递归深度 %d，停止递归分析", depth)
		return
	}

	g.Log().Infof(ctx, "开始第 %d 层递归分析，地址数量: %d", depth, len(addresses))

	// 根据递归深度调整筛选条件 todo
	//minBalance, maxBalance := m.getBalanceThresholdByDepth(depth)
	//minTxCount, maxTxCount := m.getTxCountThresholdByDepth(depth)
	minBalance := 5000.0
	maxBalance := 50000.0
	minTxCount := 20
	maxTxCount := 20000

	var activeAddresses []ActiveAddress
	var nextLevelAddresses []string

	// 并发处理地址
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, m.concurrency)

	for _, addr := range addresses {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(address string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			// 分析单个地址
			activeAddr, newAddresses := m.analyzeAddress(ctx, address, depth, minBalance, maxBalance, minTxCount, maxTxCount)

			if activeAddr.IsActive {
				mu.Lock()
				activeAddresses = append(activeAddresses, activeAddr)
				nextLevelAddresses = append(nextLevelAddresses, newAddresses...)
				mu.Unlock()
			}
		}(addr)
	}

	wg.Wait()

	g.Log().Infof(ctx, "第 %d 层递归分析完成，找到 %d 个活跃地址", depth, len(activeAddresses))

	// 保存活跃地址分析结果
	m.saveActiveAddressResults(ctx, activeAddresses)

	// 继续递归分析
	if len(nextLevelAddresses) > 0 {
		m.analyzeActiveAddressesRecursively(ctx, nextLevelAddresses, depth+1)
	}
}

// 根据递归深度获取余额阈值
func (m *ActiveMonitor) getBalanceThresholdByDepth(depth int) (float64, float64) {
	// 初始条件：5000-50000
	minBalance := 5000.0
	maxBalance := 50000.0

	// 随着递归深度增加，提高余额要求
	factor := math.Pow(1.1, float64(depth))
	return minBalance * factor, maxBalance * factor
}

// 根据递归深度获取交易数量阈值
func (m *ActiveMonitor) getTxCountThresholdByDepth(depth int) (int, int) {
	// 初始条件：20-20000
	minTxCount := 20
	maxTxCount := 20000

	// 随着递归深度增加，提高交易数量要求
	factor := math.Pow(1.1, float64(depth))
	return int(float64(minTxCount) * factor), int(float64(maxTxCount) * factor)
}

// 分析单个地址
func (m *ActiveMonitor) analyzeAddress(ctx context.Context, address string, depth int, minBalance, maxBalance float64, minTxCount, maxTxCount int) (ActiveAddress, []string) {
	activeAddr := ActiveAddress{
		Address:        address,
		RecursionDepth: depth,
	}

	// 获取地址余额
	balance, err := m.tronAPI.GetTokenBalance(ctx, address, m.usdtContract)
	if err != nil {
		g.Log().Errorf(ctx, "获取地址 %s 余额失败: %v", address, err)
		return activeAddr, nil
	}
	activeAddr.Balance = balance

	// 检查余额是否符合条件
	if balance < minBalance || balance > maxBalance {
		return activeAddr, nil
	}

	// 获取地址交易数量
	txCount, err := m.tronAPI.GetTransactionCount(ctx, address, m.usdtContract)
	if err != nil {
		g.Log().Errorf(ctx, "获取地址 %s 交易数量失败: %v", address, err)
		return activeAddr, nil
	}
	activeAddr.TxCount = txCount

	// 检查交易数量是否符合条件
	if txCount < minTxCount || txCount > maxTxCount {
		return activeAddr, nil
	}

	// 获取地址的最近USDT交易
	transactions, err := m.getRecentUSDTTransactions(ctx, address)
	if err != nil {
		g.Log().Errorf(ctx, "获取地址 %s 最近USDT交易失败: %v", address, err)
		return activeAddr, nil
	}

	// 更新最后活跃时间
	if len(transactions) > 0 {
		activeAddr.LastActiveTime = transactions[0].Timestamp
	}

	// 分析常转出地址
	frequentOutAddrs, newAddresses := m.analyzeFrequentOutAddresses(ctx, address, transactions)

	// 如果有符合条件的常转出地址，则标记为活跃
	if len(frequentOutAddrs) > 0 {
		activeAddr.IsActive = true
		activeAddr.FrequentOutAddrs = make([]string, len(frequentOutAddrs))

		for i, addr := range frequentOutAddrs {
			activeAddr.FrequentOutAddrs[i] = addr.Address

			// 生成订单并保存
			m.generateAndSaveOrder(ctx, activeAddr, addr)
		}
	}

	return activeAddr, newAddresses
}

// 分析常转出地址
func (m *ActiveMonitor) analyzeFrequentOutAddresses(ctx context.Context, sourceAddr string, transactions []ActiveTransaction) ([]FrequentOutAddress, []string) {
	// 统计转出情况
	outStats := make(map[string]*FrequentOutAddress)

	for _, tx := range transactions {
		// 只分析从源地址转出的交易
		if tx.FromAddress != sourceAddr {
			continue
		}

		// 更新或创建转出统计
		if _, exists := outStats[tx.ToAddress]; !exists {
			outStats[tx.ToAddress] = &FrequentOutAddress{
				Address:       tx.ToAddress,
				MaskedAddress: m.maskAddress(tx.ToAddress),
				LastTxTime:    tx.Timestamp,
			}
		}

		stat := outStats[tx.ToAddress]
		stat.OutCount++
		stat.TotalOutAmount += tx.Amount

		// 更新最后交易时间
		if tx.Timestamp.After(stat.LastTxTime) {
			stat.LastTxTime = tx.Timestamp
		}

		// 统计大额转出(>10000)
		if tx.Amount > 10000 {
			stat.LargeOutCount++
		}
	}

	// 计算平均转出金额并筛选符合条件的常转出地址
	var frequentOutAddrs []FrequentOutAddress
	var newAddresses []string

	for _, stat := range outStats {
		// 计算平均转出金额
		if stat.OutCount > 0 {
			stat.AvgOutAmount = stat.TotalOutAmount / float64(stat.OutCount)
		}

		// 筛选条件：
		// 1. 10000以上的交易超过5笔
		// 2. 转出笔数≥2
		// 3. 转出平均金额在1000-3000之间
		if stat.LargeOutCount >= 5 && stat.OutCount >= 2 &&
			stat.AvgOutAmount >= 1000 && stat.AvgOutAmount <= 3000 {
			frequentOutAddrs = append(frequentOutAddrs, *stat)
			newAddresses = append(newAddresses, stat.Address)
		}
	}

	// 按总转出金额排序，取转出金额最多的
	if len(frequentOutAddrs) > 1 {
		sort.Slice(frequentOutAddrs, func(i, j int) bool {
			return frequentOutAddrs[i].TotalOutAmount > frequentOutAddrs[j].TotalOutAmount
		})
	}

	return frequentOutAddrs, newAddresses
}

// 地址掩码处理
func (m *ActiveMonitor) maskAddress(address string) string {
	if len(address) <= m.prefixMaskLen+m.suffixMaskLen {
		return address
	}

	prefix := address[:m.prefixMaskLen]
	suffix := address[len(address)-m.suffixMaskLen:]

	return prefix + "*" + suffix
}

// 生成订单号
func (m *ActiveMonitor) generateOrderID(frequentOutAddr FrequentOutAddress) string {
	// 订单号规则：年月日时分+常转地址后四位
	now := time.Now()
	suffix := ""

	if len(frequentOutAddr.Address) >= 4 {
		suffix = frequentOutAddr.Address[len(frequentOutAddr.Address)-4:]
	}

	return fmt.Sprintf("%d%02d%02d%02d%02d%s",
		now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), suffix)
}

// 生成并保存订单
func (m *ActiveMonitor) generateAndSaveOrder(ctx context.Context, activeAddr ActiveAddress, frequentOutAddr FrequentOutAddress) {
	order := Order{
		OrderID:         m.generateOrderID(frequentOutAddr),
		ActiveAddress:   activeAddr.Address,
		FrequentOutAddr: frequentOutAddr.MaskedAddress,
		LastTxTime:      frequentOutAddr.LastTxTime,
		FixedAmount:     frequentOutAddr.AvgOutAmount, // 使用平均转出金额作为固定金额
		RecursionDepth:  activeAddr.RecursionDepth,
	}

	// 保存订单到数据库
	m.saveOrder(ctx, order)
}

// 保存订单到数据库
func (m *ActiveMonitor) saveOrder(ctx context.Context, order Order) {
	// TODO: 实现订单保存到数据库的逻辑
	if m.debugMode {
		g.Log().Debugf(ctx, "保存订单: %+v", order)
	}
}

// 保存活跃地址分析结果
func (m *ActiveMonitor) saveActiveAddressResults(ctx context.Context, activeAddresses []ActiveAddress) {
	// TODO: 实现活跃地址分析结果保存到数据库的逻辑
	if m.debugMode {
		g.Log().Debugf(ctx, "保存活跃地址分析结果: %d 条", len(activeAddresses))
	}
}
