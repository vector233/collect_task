package cmd

import (
	"context"
	"fmt"

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
	maxDepth := g.Cfg().MustGet(ctx, "tron.maxDepth", 3).Int()

	// 记录总共处理的地址数量
	var totalProcessed, totalInserted int

	// 记录已处理地址，防止重复处理
	processedAddresses := make(map[string]struct{})

	// 开始递归处理
	err = processAddressRecursivelyD(ctx, address, 0, maxDepth, processedAddresses, &totalProcessed, &totalInserted)
	if err != nil {
		return err
	}

	fmt.Printf("[完成] 总计处理 %d 个地址, 新增 %d 个地址\n", totalProcessed, totalInserted)
	return nil
}

// 递归处理地址及其交易记录中的地址
func processAddressRecursivelyD(
	ctx context.Context,
	address string,
	currentDepth,
	maxDepth int,
	processedAddresses map[string]struct{},
	totalProcessed,
	totalInserted *int,
) error {
	// 检查是否已处理过该地址
	if _, exists := processedAddresses[address]; exists {
		return nil
	}

	// 标记该地址为已处理
	processedAddresses[address] = struct{}{}

	// 获取该地址的交易记录中的所有地址
	fmt.Printf("深度 %d/%d: 正在处理地址: %s\n", currentDepth+1, maxDepth, address)
	addresses, err := fetchAddresses(ctx, address)
	if err != nil {
		return fmt.Errorf("获取地址 %s 的交易记录失败: %v", address, err)
	}

	*totalProcessed += len(addresses)
	fmt.Printf("深度 %d/%d: 地址 %s 找到 %d 个相关地址\n",
		currentDepth+1, maxDepth, address, len(addresses))

	// 批量查询数据库中已存在的地址
	existingAddresses, err := batchCheckAddressesD(ctx, addresses)
	if err != nil {
		return fmt.Errorf("批量查询地址失败: %v", err)
	}

	// 找出需要插入的新地址
	var newAddresses []string
	for _, addr := range addresses {
		if _, exists := existingAddresses[addr]; !exists {
			newAddresses = append(newAddresses, addr)
		}
	}

	fmt.Printf("深度 %d/%d: 已存在 %d 个地址, 需要插入 %d 个新地址\n",
		currentDepth+1, maxDepth, len(existingAddresses), len(newAddresses))

	// 如果有新地址需要插入
	if len(newAddresses) > 0 {
		// 批量插入新地址
		if err := batchInsertAddressesD(ctx, newAddresses); err != nil {
			return fmt.Errorf("批量插入地址失败: %v", err)
		}

		*totalInserted += len(newAddresses)
	}

	// 如果未达到最大深度，继续递归处理新地址
	if currentDepth < maxDepth-1 && len(newAddresses) > 0 {
		// 限制每层递归处理的地址数量，避免爆炸式增长
		maxAddressesPerLevel := g.Cfg().MustGet(ctx, "tron.maxAddressesPerLevel", 10).Int()
		fmt.Printf("限制每层递归处理的地址数量为 %d 个\n", maxAddressesPerLevel)
		processCount := len(newAddresses)
		if processCount > maxAddressesPerLevel {
			processCount = maxAddressesPerLevel
			fmt.Printf("深度 %d/%d: 限制处理地址数量为 %d 个\n",
				currentDepth+2, maxDepth, processCount)
		}

		for i := 0; i < processCount; i++ {
			fmt.Printf("深度 %d/%d:  总共 %d 个，当前处理第 %d 个\n",
				currentDepth+2, maxDepth, processCount, i)
			err := processAddressRecursivelyD(
				ctx,
				newAddresses[i],
				currentDepth+1,
				maxDepth,
				processedAddresses,
				totalProcessed,
				totalInserted,
			)
			if err != nil {
				fmt.Printf("处理地址 %s 时出错: %v\n", newAddresses[i], err)
				// 继续处理其他地址，不中断整个流程
				continue
			}
		}
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
		addr := record["from_address"].String()
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
