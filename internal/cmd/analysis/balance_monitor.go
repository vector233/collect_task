package analysis

import (
	"context"
	"fmt"
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcron"
	"github.com/gogf/gf/v2/os/gctx"
	"github.com/gogf/gf/v2/os/gtime"
)

// BalanceMonitor 余额监控器
type BalanceMonitor struct {
	tronAPI      *TronAPI
	usdtContract string
	batchSize    int
	ctx          context.Context
	tableName    string // 表名
	concurrency  int    // 并发数
}

// NewBalanceMonitor 创建余额监控器
func NewBalanceMonitor(tronAPI *TronAPI, usdtContract string) *BalanceMonitor {
	return &BalanceMonitor{
		tronAPI:      tronAPI,
		usdtContract: usdtContract,
		batchSize:    200, // 批量处理数
		ctx:          gctx.New(),
		tableName:    "t_order_from_address", // 默认表
		concurrency:  200,                    // 默认并发
	}
}

// SetTable 设置表名
func (m *BalanceMonitor) SetTable(tableName string) {
	if tableName != "" {
		m.tableName = tableName
	}
	g.Log().Infof(m.ctx, "余额监控器配置更新: 使用表=%s", m.tableName)
}

// SetConcurrency 设置并发数
func (m *BalanceMonitor) SetConcurrency(concurrency int) {
	if concurrency > 0 {
		m.concurrency = concurrency
	}
	g.Log().Infof(m.ctx, "余额监控器配置更新: 并发数=%d", m.concurrency)
}

// SetBatchSize 设置批处理大小
func (m *BalanceMonitor) SetBatchSize(batchSize int) {
	if batchSize > 0 {
		m.batchSize = batchSize
	}
	g.Log().Infof(m.ctx, "余额监控器配置更新: 批处理大小=%d", m.batchSize)
}

// 获取表对应的字段配置
func (m *BalanceMonitor) getTableFields() (addressField, balanceField, timeField string) {
	// 根据表名返回字段
	switch m.tableName {
	case "t_order_from_address":
		return "from_address", "balance", "query_time"
	case "t_address_order":
		return "address", "balance", "query_time"
	default:
		// 默认字段名
		return "from_address", "balance", "query_time"
	}
}

// StartMonitor 启动定时任务
// pattern: cron表达式，如 "0 */30 * * * *" 每30分钟执行一次
func (m *BalanceMonitor) StartMonitor(pattern string) error {
	_, err := gcron.Add(m.ctx, pattern, func(ctx context.Context) {
		m.UpdateAllAddressesBalance(ctx)
	}, "UpdateAddressBalance")

	if err != nil {
		return fmt.Errorf("启动余额监控定时任务失败: %v", err)
	}

	g.Log().Infof(m.ctx, "余额监控定时任务已启动，执行周期: %s", pattern)
	return nil
}

type AddressField struct {
	Address string `json:"address"`
}

type BalanceResult struct {
	Address string
	Balance float64
	Error   error
}

// UpdateAllAddressesBalance 刷新所有地址余额
func (m *BalanceMonitor) UpdateAllAddressesBalance(ctx context.Context) {
	g.Log().Info(ctx, "开始更新地址余额...")

	// 获取当前表的字段配置
	addressField, balanceField, timeField := m.getTableFields()

	// 创建全局结果通道，设置合理的缓冲区大小
	resultChan := make(chan BalanceResult, m.concurrency*2)

	// 启动消费者协程，处理查询结果并更新数据库
	var consumerWg sync.WaitGroup
	consumerWg.Add(1)

	// 用于统计成功处理的地址数量
	successCountChan := make(chan int, 1)

	go func() {
		defer consumerWg.Done()
		successCount := m.collectAndUpdateBalances(ctx, resultChan, addressField, balanceField, timeField)
		successCountChan <- successCount
	}()

	// 分页查询参数
	pageSize := 500
	page := 1
	totalProcessed := 0

	// 用于等待所有查询协程完成
	var producerWg sync.WaitGroup

	// 限制同时处理的页数
	pageSemaphore := make(chan struct{}, 5) // 最多同时处理5页

	for {
		// 分页查询地址
		var addresses []AddressField
		err := g.DB().Model(m.tableName).
			Ctx(ctx).
			Fields(fmt.Sprintf("%s as address", addressField)).
			Page(page, pageSize).
			Scan(&addresses)

		if err != nil {
			g.Log().Errorf(ctx, "获取监控地址列表失败: %v", err)
			break
		}

		// 如果没有查询到数据，说明已经处理完所有地址
		if len(addresses) == 0 {
			break
		}

		g.Log().Infof(ctx, "正在处理第 %d 页地址，本页共 %d 个地址", page, len(addresses))

		// 处理当前页的地址
		producerWg.Add(1)
		pageSemaphore <- struct{}{} // 获取页面处理信号量
		go func(addrs []AddressField, pageNum int) {
			defer producerWg.Done()
			defer func() { <-pageSemaphore }() // 释放页面处理信号量

			processed := m.fetchBalancesConcurrently(ctx, addrs, resultChan)
			g.Log().Infof(ctx, "第 %d 页地址查询完成，共处理 %d 个地址", pageNum, processed)
		}(addresses, page)

		// 更新统计信息
		totalProcessed += len(addresses)
		page++
	}

	// 等待所有生产者完成
	producerWg.Wait()

	// 关闭结果通道，通知消费者没有更多数据
	close(resultChan)

	// 等待消费者处理完所有结果
	consumerWg.Wait()

	// 获取成功处理的数量
	successCount := <-successCountChan
	close(successCountChan)

	g.Log().Infof(ctx, "地址余额更新完成，共处理 %d 个地址，成功更新 %d 个", totalProcessed, successCount)
}

// fetchBalancesConcurrently 并发获取地址余额
func (m *BalanceMonitor) fetchBalancesConcurrently(ctx context.Context, addresses []AddressField,
	resultChan chan<- BalanceResult) int {

	// 创建并发控制
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, m.concurrency)

	// 并发查询每个地址的余额
	validAddressCount := 0
	for _, addr := range addresses {
		address := addr.Address
		if address == "" {
			continue
		}
		validAddressCount++

		wg.Add(1)
		go func(addr string) {
			defer wg.Done()

			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			// 查询USDT余额
			balance, err := m.tronAPI.GetTokenBalance(ctx, addr, m.usdtContract)

			// 发送结果到通道
			resultChan <- BalanceResult{
				Address: addr,
				Balance: balance,
				Error:   err,
			}
		}(address)
	}

	// 等待所有查询完成
	wg.Wait()
	return validAddressCount
}

// collectAndUpdateBalances 收集结果并批量更新数据库
func (m *BalanceMonitor) collectAndUpdateBalances(ctx context.Context, resultChan <-chan BalanceResult,
	addressField, balanceField, timeField string) int {

	successCount := 0
	batch := make([]BalanceResult, 0, m.batchSize)
	var wg sync.WaitGroup

	// 用于跟踪异步写入的结果
	resultCountChan := make(chan int, 100)

	// 用于等待计数协程完成的通道
	countDone := make(chan struct{})

	// 启动一个协程来收集写入结果的计数
	go func() {
		for count := range resultCountChan {
			successCount += count
		}
		close(countDone) // 通知计数完成
	}()

	for result := range resultChan {
		if result.Error != nil {
			g.Log().Errorf(ctx, "查询地址 %s 余额失败: %v", result.Address, result.Error)
			continue
		}

		g.Log().Debugf(ctx, "地址 %s 余额查询结果: %.6f USDT", result.Address, result.Balance)
		batch = append(batch, result)

		// 达到批量大小，异步执行批量写入
		if len(batch) >= m.batchSize {
			// 创建当前批次的副本，避免数据竞争
			batchCopy := make([]BalanceResult, len(batch))
			copy(batchCopy, batch)

			wg.Add(1)
			go func(batchData []BalanceResult) {
				defer wg.Done()
				count := m.flushBalanceBatch(ctx, batchData, addressField, balanceField, timeField)
				resultCountChan <- count
			}(batchCopy)

			batch = batch[:0] // 清空批次
		}
	}

	// 处理剩余的结果
	if len(batch) > 0 {
		wg.Add(1)
		go func(batchData []BalanceResult) {
			defer wg.Done()
			count := m.flushBalanceBatch(ctx, batchData, addressField, balanceField, timeField)
			resultCountChan <- count
		}(batch)
	}

	// 等待所有异步写入操作完成
	wg.Wait()
	close(resultCountChan)

	// 等待计数协程完成
	<-countDone

	return successCount
}

// flushBalanceBatch 将一批余额结果写入数据库
func (m *BalanceMonitor) flushBalanceBatch(ctx context.Context, batch []BalanceResult,
	addressField, balanceField, timeField string) int {

	if len(batch) == 0 {
		return 0
	}

	m.batchUpdateBalances(ctx, batch, addressField, balanceField, timeField)
	return len(batch)
}

// 批量更新余额
func (m *BalanceMonitor) batchUpdateBalances(ctx context.Context, results []BalanceResult,
	addressField, balanceField, timeField string) {

	if len(results) == 0 {
		return
	}

	// 构建批量更新的数据
	batchData := make([]g.Map, 0, len(results))
	addresses := make([]string, 0, len(results))

	for _, result := range results {
		batchData = append(batchData, g.Map{
			addressField: result.Address,
			balanceField: result.Balance,
			timeField:    gtime.Now(),
		})
		addresses = append(addresses, result.Address)
	}

	// 使用批量更新
	_, err := g.DB().Ctx(ctx).Model(m.tableName).
		Data(batchData).
		WherePri(addressField).
		WhereIn(addressField, addresses).
		Save()

	if err != nil {
		g.Log().Errorf(ctx, "批量更新余额失败: %v", err)
	} else {
		g.Log().Infof(ctx, "成功批量更新 %d 个地址的余额", len(results))
	}
}

// UpdateSingleAddressBalance 更新单个地址余额
func (m *BalanceMonitor) UpdateSingleAddressBalance(ctx context.Context, address string) (float64, error) {
	// 获取当前表的字段配置
	addressField, balanceField, timeField := m.getTableFields()

	// 查询USDT余额
	balance, err := m.tronAPI.GetTokenBalance(ctx, address, m.usdtContract)
	if err != nil {
		return 0, fmt.Errorf("查询地址 %s 余额失败: %v", address, err)
	}

	// 更新数据库
	data := g.Map{
		balanceField: balance,
		timeField:    gtime.Now(),
	}

	_, err = g.DB().Model(m.tableName).Ctx(ctx).Data(data).Where(addressField, address).
		Update()

	if err != nil {
		return balance, fmt.Errorf("更新地址 %s 余额失败: %v", address, err)
	}

	return balance, nil
}
