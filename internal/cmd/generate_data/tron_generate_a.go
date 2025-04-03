package generate_data

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gtime"

	"tron-lion/internal/dao"
	"tron-lion/internal/model/entity"
)

var (
	TronGenerateA = gcmd.Command{
		Name:  "gen_a",
		Usage: "tron gen_a count prefix suffix",
		Brief: "生成前X后Y格式的地址匹配规则并插入TOrderAddressRecordResult表",
		Func:  runTronGenerateA,
	}
)

func runTronGenerateA(ctx context.Context, parser *gcmd.Parser) (err error) {
	// 获取生成数量
	count := g.Cfg().MustGet(ctx, "tron.count", 10000).Int()
	if count <= 0 {
		count = 10
	}

	// 获取批处理大小，默认为1000
	batchSize := g.Cfg().MustGet(ctx, "tron.batchSize", 1000).Int()
	if batchSize <= 0 {
		batchSize = 1000
	}

	fmt.Printf("准备生成 %d 条地址，批处理大小: %d\n", count, batchSize)

	// 初始化随机数
	rand.Seed(time.Now().UnixNano())

	// 计算需要处理的批次数
	batchCount := (count + batchSize - 1) / batchSize
	totalInserted := 0

	for i := 0; i < batchCount; i++ {
		// 计算当前批次需要生成的记录数
		currentBatchSize := batchSize
		if i == batchCount-1 && count%batchSize != 0 {
			currentBatchSize = count % batchSize
		}

		fmt.Printf("处理批次 %d/%d，生成 %d 条记录\n", i+1, batchCount, currentBatchSize)

		// 批量生成记录
		records := make([]*entity.TOrderAddressRecordResult, 0, currentBatchSize)
		for j := 0; j < currentBatchSize; j++ {
			records = append(records, &entity.TOrderAddressRecordResult{
				FromAddressPart: genFromAddressPart(ctx),
				CreateTime:      gtime.Now(),
			})
		}

		// 批量插入
		result, err := dao.TOrderAddressRecordResult.Ctx(ctx).Insert(records)
		if err != nil {
			return fmt.Errorf("插入数据库失败 (批次 %d/%d): %v", i+1, batchCount, err)
		}

		// 获取插入的记录数
		affected, _ := result.RowsAffected()
		totalInserted += int(affected)

		fmt.Printf("批次 %d/%d 完成，成功插入 %d 条记录\n", i+1, batchCount, affected)
	}

	fmt.Printf("所有批次处理完成，总共成功插入 %d 条地址\n", totalInserted)
	return nil
}
