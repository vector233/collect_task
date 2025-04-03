package generate_data

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gtime"

	"tron-lion/internal/dao"
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

	// 递归深度限制
	maxDepth := g.Cfg().MustGet(ctx, "tron.maxDepth", 100).Int()

	var totalProcessed, totalInserted int
	processedAddresses := make(map[string]struct{})
	var mu sync.Mutex

	// 开始处理
	err = processAddressesWithConcurrencyCommon(
		ctx,
		[]string{address},
		0,
		maxDepth,
		processedAddresses,
		&totalProcessed,
		&totalInserted,
		&mu,
		insertOrIgnoreAddressesD,
	)
	if err != nil {
		return err
	}

	fmt.Printf("[完成] 总计处理 %d 个地址, 新增 %d 个地址\n", totalProcessed, totalInserted)
	return nil
}

// 插入地址到TOrderToAddressRecord表
func insertOrIgnoreAddressesD(ctx context.Context, addresses []string) (sql.Result, error) {
	if len(addresses) == 0 {
		return nil, nil
	}

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
