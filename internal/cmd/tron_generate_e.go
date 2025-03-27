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
	TronGenerateE = gcmd.Command{
		Name:  "gen_e",
		Usage: "tron gen_e",
		Brief: "生成数据并插入到TOrderFromAddress表",
		Func:  runTronGenerateE,
	}
)

func runTronGenerateE(ctx context.Context, parser *gcmd.Parser) (err error) {
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

	// 开始递归处理
	err = processAddressesWithConcurrency(ctx, []string{address}, 0, maxDepth, processedAddresses, &totalProcessed, &totalInserted, &mu)
	if err != nil {
		return err
	}

	fmt.Printf("[完成] 总计处理 %d 个地址, 新增 %d 个地址\n", totalProcessed, totalInserted)
	return nil
}

// 并发处理一批地址
func processAddressesWithConcurrency(
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

			// 直接使用 INSERT ON DUPLICATE KEY UPDATE 操作
			// 对于已存在的地址会忽略，不存在的会插入
			result, err := insertOrIgnoreAddresses(ctx, addresses)
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
			// 注意：这里我们不再区分新旧地址，而是将所有地址都加入队列
			// 因为我们已经不再单独查询哪些是新地址了
			// 但是由于我们在处理前会检查地址是否已处理过，所以不会重复处理
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
		return processAddressesWithConcurrency(
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
func batchCheckAddresses(ctx context.Context, addresses []string) (map[string]struct{}, error) {
	if len(addresses) == 0 {
		return make(map[string]struct{}), nil
	}

	// 查询数据库中已存在的地址
	records, err := dao.TOrderFromAddress.Ctx(ctx).
		Where(dao.TOrderFromAddress.Columns().FromAddress, addresses).
		Fields(dao.TOrderFromAddress.Columns().FromAddress).
		All()

	if err != nil {
		return nil, fmt.Errorf("查询数据库失败: %v", err)
	}

	// 将查询结果转换为map便于快速查找
	existingAddresses := make(map[string]struct{}, len(records))
	for _, record := range records {
		addr := record["from_address"].String()
		existingAddresses[addr] = struct{}{}
	}

	return existingAddresses, nil
}

// 批量插入地址到数据库
func batchInsertAddresses(ctx context.Context, addresses []string) error {
	if len(addresses) == 0 {
		return nil
	}

	// 准备批量插入的数据
	batch := make([]map[string]interface{}, 0, len(addresses))
	now := gtime.Now()

	for _, addr := range addresses {
		batch = append(batch, map[string]interface{}{
			dao.TOrderFromAddress.Columns().FromAddress: addr,
			dao.TOrderFromAddress.Columns().CreateTime:  now,
		})
	}

	// 执行批量插入
	_, err := dao.TOrderFromAddress.Ctx(ctx).
		Data(batch).
		Batch(200).
		Insert()

	if err != nil {
		return fmt.Errorf("批量插入数据库失败: %v", err)
	}

	return nil
}

// 使用 INSERT ON DUPLICATE KEY UPDATE 插入地址
// 对于已存在的地址会忽略，不存在的会插入
func insertOrIgnoreAddresses(ctx context.Context, addresses []string) (sql.Result, error) {
	if len(addresses) == 0 {
		return nil, nil
	}

	// 准备批量插入的数据
	batch := make([]map[string]interface{}, 0, len(addresses))
	now := gtime.Now()

	for _, addr := range addresses {
		batch = append(batch, map[string]interface{}{
			dao.TOrderFromAddress.Columns().FromAddress: addr,
			dao.TOrderFromAddress.Columns().CreateTime:  now,
		})
	}

	// 执行带有 ON DUPLICATE KEY UPDATE 的插入
	// 这里我们使用 create_time=create_time 表示不更新任何字段
	// 这样对于已存在的记录不会有任何变化
	return dao.TOrderFromAddress.Ctx(ctx).
		Data(batch).
		Batch(500).
		InsertIgnore()
}
