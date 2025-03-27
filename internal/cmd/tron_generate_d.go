package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gtime"

	"github.com/bivdex/tron-lion/internal/dao"
)

var (
	TronGenerateD = gcmd.Command{
		Name:  "gen_d",
		Usage: "tron gen_d",
		Brief: "生成数据并插入到TOrderToAddressRecord表",
		Func:  runTronGenerateD,
	}
)

func runTronGenerateD(ctx context.Context, parser *gcmd.Parser) (err error) {
	// 从配置读取初始地址
	address := g.Cfg().MustGet(ctx, "tron.address").String()
	if address == "" {
		return fmt.Errorf("未配置波场地址")
	}

	fmt.Printf("[开始] 处理初始地址: %s\n", address)

	// 递归深度限制，避免无限递归
	maxDepth := g.Cfg().MustGet(ctx, "tron.maxDepth", 100).Int()

	// 记录总共处理的地址数量
	var totalProcessed, totalInserted int

	// 记录已处理地址，防止重复处理
	processedAddresses := make(map[string]struct{})
	// 使用互斥锁保护 processedAddresses 和计数器
	var mu sync.Mutex

	// 开始并发处理
	err = processAddressesWithConcurrencyD(ctx, []string{address}, 0, maxDepth, processedAddresses, &totalProcessed, &totalInserted, &mu)
	if err != nil {
		return err
	}

	fmt.Printf("[完成] 总计处理 %d 个地址, 新增 %d 个地址\n", totalProcessed, totalInserted)
	return nil
}

// 并发处理一批地址
func processAddressesWithConcurrencyD(
	ctx context.Context,
	addresses []string,
	currentDepth,
	maxDepth int,
	processedAddresses map[string]struct{},
	totalProcessed,
	totalInserted *int,
	mu *sync.Mutex,
) error {
	if currentDepth >= maxDepth {
		return nil
	}

	// 从配置中读取并发数
	maxConcurrency := g.Cfg().MustGet(ctx, "tron.maxConcurrency", 50).Int()

	fmt.Printf("深度 %d/%d: 开始并发处理 %d 个地址 (并发数: %d)\n",
		currentDepth+1, maxDepth, len(addresses), maxConcurrency)

	// 创建并发控制通道
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	// 收集下一层需要处理的地址
	var nextLevelAddresses []string
	var nextLevelMu sync.Mutex

	for _, addr := range addresses {
		// 检查是否已处理过该地址
		mu.Lock()
		if _, exists := processedAddresses[addr]; exists {
			mu.Unlock()
			continue
		}
		// 标记为已处理
		processedAddresses[addr] = struct{}{}
		mu.Unlock()

		// 并发控制
		semaphore <- struct{}{}
		wg.Add(1)

		// 启动goroutine处理地址
		go func(address string) {
			defer func() {
				<-semaphore // 释放信号量
				wg.Done()
			}()

			// 获取该地址的交易记录中的所有地址
			fmt.Printf("深度 %d/%d: 正在处理地址: %s\n", currentDepth+1, maxDepth, address)
			addresses, err := fetchAddresses(ctx, address)
			if err != nil {
				fmt.Printf("获取地址 %s 的交易记录失败: %v\n", address, err)
				return
			}

			mu.Lock()
			*totalProcessed += len(addresses)
			mu.Unlock()

			fmt.Printf("深度 %d/%d: 地址 %s 找到 %d 个相关地址\n",
				currentDepth+1, maxDepth, address, len(addresses))

			result, err := insertOrIgnoreAddressesD(ctx, addresses)
			if err != nil {
				fmt.Printf("插入地址失败: %v\n", err)
				return
			}

			// 获取实际插入的记录数
			insertedCount, err := result.RowsAffected()
			if err != nil {
				fmt.Printf("获取插入结果数量失败: %v\n", err)
				return
			}

			mu.Lock()
			*totalInserted += int(insertedCount)
			mu.Unlock()

			fmt.Printf("深度 %d/%d: 地址 %s 处理完成，新增 %d 个地址\n",
				currentDepth+1, maxDepth, address, insertedCount)

			// 将所有地址添加到下一层处理队列
			if currentDepth < maxDepth-1 {
				nextLevelMu.Lock()
				nextLevelAddresses = append(nextLevelAddresses, addresses...)
				nextLevelMu.Unlock()
			}
		}(addr)
	}

	// 等待所有goroutine完成
	wg.Wait()

	// 如果未达到最大深度，继续处理下一层地址
	if currentDepth < maxDepth-1 && len(nextLevelAddresses) > 0 {
		// 限制每层处理的地址数量，避免爆炸式增长
		maxAddressesPerLevel := g.Cfg().MustGet(ctx, "tron.maxAddressesPerLevel", 1000).Int()
		fmt.Printf("深度 %d/%d: 下一层有 %d 个地址待处理，限制为 %d 个\n",
			currentDepth+1, maxDepth, len(nextLevelAddresses), maxAddressesPerLevel)

		if len(nextLevelAddresses) > maxAddressesPerLevel {
			nextLevelAddresses = nextLevelAddresses[:maxAddressesPerLevel]
		}

		// 递归处理下一层地址
		return processAddressesWithConcurrencyD(
			ctx,
			nextLevelAddresses,
			currentDepth+1,
			maxDepth,
			processedAddresses,
			totalProcessed,
			totalInserted,
			mu,
		)
	}

	return nil
}

// 批量检查地址是否存在于数据库
func batchCheckAddressesD(ctx context.Context, addresses []string) (map[string]struct{}, error) {
	if len(addresses) == 0 {
		return make(map[string]struct{}), nil
	}

	// 查询数据库中已存在的地址
	records, err := dao.TOrderToAddressRecord.Ctx(ctx).
		Where(dao.TOrderToAddressRecord.Columns().ToAddress, addresses).
		Fields(dao.TOrderToAddressRecord.Columns().ToAddress).
		All()

	if err != nil {
		return nil, fmt.Errorf("查询数据库失败: %v", err)
	}

	// 将查询结果转换为map便于快速查找
	existingAddresses := make(map[string]struct{}, len(records))
	for _, record := range records {
		addr := record["to_address"].String() // 修正字段名
		existingAddresses[addr] = struct{}{}
	}

	return existingAddresses, nil
}

// 批量插入地址到数据库
func batchInsertAddressesD(ctx context.Context, addresses []string) error {
	if len(addresses) == 0 {
		return nil
	}

	// 准备批量插入的数据
	batch := make([]map[string]interface{}, 0, len(addresses))
	now := gtime.Now()

	for _, addr := range addresses {
		batch = append(batch, map[string]interface{}{
			dao.TOrderToAddressRecord.Columns().FromAddressPart: genFromAddressPart(ctx),
			dao.TOrderToAddressRecord.Columns().ToAddress:       addr,
			dao.TOrderToAddressRecord.Columns().CreateTime:      now,
		})
	}

	// 执行批量插入
	_, err := dao.TOrderToAddressRecord.Ctx(ctx).
		Data(batch).
		Batch(200).
		Insert()

	if err != nil {
		return fmt.Errorf("批量插入数据库失败: %v", err)
	}

	return nil
}

// 使用 INSERT IGNORE 插入地址
func insertOrIgnoreAddressesD(ctx context.Context, addresses []string) (sql.Result, error) {
	if len(addresses) == 0 {
		return nil, nil
	}

	// 准备批量插入的数据
	batch := make([]map[string]interface{}, 0, len(addresses))
	now := gtime.Now()

	for _, addr := range addresses {
		batch = append(batch, map[string]interface{}{
			dao.TOrderToAddressRecord.Columns().FromAddressPart: genFromAddressPart(ctx),
			dao.TOrderToAddressRecord.Columns().ToAddress:       addr,
			dao.TOrderToAddressRecord.Columns().CreateTime:      now,
		})
	}

	return dao.TOrderToAddressRecord.Ctx(ctx).
		Data(batch).
		Batch(500).
		InsertIgnore()
}

func genFromAddressPart(ctx context.Context) string {
	prefix := g.Cfg().MustGet(ctx, "tron.prefix").Int()
	suffix := g.Cfg().MustGet(ctx, "tron.suffix").Int()
	if prefix <= 0 {
		prefix = 3 // 默认前缀长度为3
	}
	if suffix <= 0 {
		suffix = 4 // 默认后缀长度为4
	}
	// 生成前缀，确保首字母是T
	prefixStr := "T" + generateRandomString(prefix-1)
	// 生成后缀
	suffixStr := generateRandomString(suffix)
	// 组合成匹配规则
	pattern := prefixStr + "*" + suffixStr
	return pattern
}
