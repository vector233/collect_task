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
		batchSize:    50, // 批量处理数
		ctx:          gctx.New(),
		tableName:    "t_order_from_address", // 默认表
		concurrency:  15,                     // 默认并发
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

// UpdateAllAddressesBalance 刷新所有地址余额
func (m *BalanceMonitor) UpdateAllAddressesBalance(ctx context.Context) {
	g.Log().Info(ctx, "开始更新地址余额...")

	// 获取当前表的字段配置
	addressField, _, _ := m.getTableFields()

	// 分页查询参数
	pageSize := m.batchSize
	page := 1
	totalProcessed := 0

	// 创建并发控制通道
	semaphore := make(chan struct{}, m.concurrency)
	var wg sync.WaitGroup

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
			return
		}

		// 如果没有查询到数据，说明已经处理完所有地址
		if len(addresses) == 0 {
			break
		}

		g.Log().Infof(ctx, "正在处理第 %d 页地址，本页共 %d 个地址", page, len(addresses))

		// 并发处理当前批次的地址
		wg.Add(1)
		semaphore <- struct{}{} // 获取信号量
		go func(pageAddresses []AddressField, pageNum int) {
			defer wg.Done()
			defer func() { <-semaphore }() // 释放信号量

			g.Log().Infof(ctx, "开始处理第 %d 页地址", pageNum)
			m.processBatch(ctx, pageAddresses)
			g.Log().Infof(ctx, "第 %d 页地址处理完成", pageNum)
		}(addresses, page)

		// 更新统计信息
		totalProcessed += len(addresses)
		page++
	}

	// 等待所有协程完成
	wg.Wait()
	close(semaphore)

	g.Log().Infof(ctx, "地址余额更新完成，共处理 %d 个地址", totalProcessed)
}

// processBatch 处理一批地址
func (m *BalanceMonitor) processBatch(ctx context.Context, addresses []AddressField) {

	for _, addr := range addresses {
		address := addr.Address
		if address == "" {
			continue
		}

		// 调用单个地址余额更新方法
		balance, err := m.UpdateSingleAddressBalance(ctx, address)
		if err != nil {
			g.Log().Errorf(ctx, "更新地址 %s 余额失败: %v", address, err)
			continue
		}

		g.Log().Debugf(ctx, "地址 %s 余额更新为 %.6f USDT", address, balance)
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
