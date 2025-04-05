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
	concurrency       int  // 并发数
	lookbackDays      int  // 交易回溯天数
	maxRecursionDepth int  // 最大递归深度
	prefixMaskLen     int  // 地址前缀掩码长度
	suffixMaskLen     int  // 地址后缀掩码长度
	batchSize         int  // 批处理大小
	debugMode         bool // 调试模式
}

// ActiveTransaction 活跃度分析用交易记录结构
type ActiveTransaction struct {
	TxID           string    // 交易ID
	BlockNum       int64     // 区块号
	Timestamp      time.Time // 交易时间
	FromAddress    string    // 转出地址
	ToAddress      string    // 转入地址
	Amount         float64   // 交易金额
	TokenType      string    // 代币类型
	ContractAddr   string    // 合约地址
	Confirmed      bool      // 是否确认
	TransactionFee float64   // 交易费用
}

// 常转出地址结构
type FrequentOutAddress struct {
	Address        string    // 地址
	MaskedAddress  string    // 掩码后地址
	OutCount       int       // 转出次数
	TotalOutAmount float64   // 总转出金额
	AvgOutAmount   float64   // 平均转出金额
	LastTxTime     time.Time // 最后交易时间
	LargeOutCount  int       // 大额转出次数(>10000)
}

// 订单结构
type Order struct {
	OrderID         string    // 订单号
	ActiveAddress   string    // 活跃地址
	FrequentOutAddr string    // 常转出地址(掩码后)
	LastTxTime      time.Time // 最近交易时间
	FixedAmount     float64   // 固定金额
	RecursionDepth  int       // 递归深度
}

// 创建活跃度监控器
func NewActiveMonitor(tronAPI *TronAPI, usdtContract string) *ActiveMonitor {
	return &ActiveMonitor{
		tronAPI:           tronAPI,
		usdtContract:      usdtContract,
		ctx:               gctx.New(),
		concurrency:       50,    // 默认并发数
		lookbackDays:      30,    // 默认回溯30天
		maxRecursionDepth: 100,   // 最大递归深度
		prefixMaskLen:     3,     // 默认前3位
		suffixMaskLen:     4,     // 默认后4位
		batchSize:         100,   // 批处理大小
		debugMode:         false, // 默认关闭调试
	}
}

// 设置并发数
func (m *ActiveMonitor) SetConcurrency(concurrency int) {
	if concurrency > 0 {
		m.concurrency = concurrency
	}
	g.Log().Infof(m.ctx, "活跃度监控器配置更新: 并发数=%d", m.concurrency)
}

// 设置回溯天数
func (m *ActiveMonitor) SetLookbackDays(days int) {
	if days > 0 {
		m.lookbackDays = days
	}
	g.Log().Infof(m.ctx, "活跃度监控器配置更新: 回溯天数=%d", m.lookbackDays)
}

// 设置最大递归深度
func (m *ActiveMonitor) SetMaxRecursionDepth(depth int) {
	if depth > 0 {
		m.maxRecursionDepth = depth
	}
	g.Log().Infof(m.ctx, "活跃度监控器配置更新: 最大递归深度=%d", m.maxRecursionDepth)
}

// 设置地址掩码长度
func (m *ActiveMonitor) SetAddressMaskLength(prefix, suffix int) {
	if prefix > 0 {
		m.prefixMaskLen = prefix
	}
	if suffix > 0 {
		m.suffixMaskLen = suffix
	}
	g.Log().Infof(m.ctx, "活跃度监控器配置更新: 地址掩码长度=前%d后%d", m.prefixMaskLen, m.suffixMaskLen)
}

// 设置调试模式
func (m *ActiveMonitor) SetDebugMode(debug bool) {
	m.debugMode = debug
	g.Log().Infof(m.ctx, "活跃度监控器配置更新: 调试模式=%v", m.debugMode)
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
	latestBlock, err := m.tronAPI.GetLatestBlock(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取最新区块失败: %v", err)
	}

	g.Log().Infof(ctx, "获取到最新区块: %d, 包含 %d 笔交易", latestBlock.BlockNumber, len(latestBlock.Transactions))

	// 获取区块中的交易详情
	var transactions []ActiveTransaction
	var wg sync.WaitGroup
	var mu sync.Mutex
	semaphore := make(chan struct{}, m.concurrency)

	for _, txID := range latestBlock.Transactions {
		wg.Add(1)
		semaphore <- struct{}{}

		go func(txID string) {
			defer wg.Done()
			defer func() { <-semaphore }()

			// 使用已实现的GetTransaction方法获取交易详情
			tx, err := m.tronAPI.GetTransaction(ctx, txID)
			if err != nil {
				g.Log().Warningf(ctx, "获取交易 %s 详情失败: %v", txID, err)
				return
			}

			// 过滤非USDT交易
			if tx.ContractAddress != m.usdtContract {
				return
			}

			// 过滤金额不在1000-5000之间的交易
			if tx.Amount < 1000 || tx.Amount > 5000 {
				return
			}

			// 转换为活跃度分析用交易格式
			activeTx := ActiveTransaction{
				TxID:           tx.TxID,
				BlockNum:       tx.BlockNumber,
				Timestamp:      time.Unix(tx.BlockTimestamp/1000, 0),
				FromAddress:    tx.From,
				ToAddress:      tx.To,
				Amount:         tx.Amount,
				TokenType:      tx.TokenName,
				ContractAddr:   tx.ContractAddress,
				Confirmed:      tx.Confirmed,
				TransactionFee: tx.Fee,
			}

			mu.Lock()
			transactions = append(transactions, activeTx)
			mu.Unlock()
		}(txID)
	}

	wg.Wait()
	return transactions, nil
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
		if tx.Amount < 1000 || tx.Amount > 5000 {
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
			Confirmed:      tx.Confirmed,
			TransactionFee: 0, // TRC20Transaction中没有交易费用
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

	// 根据递归深度调整筛选条件
	minBalance, maxBalance := m.getBalanceThresholdByDepth(depth)
	minTxCount, maxTxCount := m.getTxCountThresholdByDepth(depth)

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
