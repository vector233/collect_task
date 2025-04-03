package generate_data

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gtime"

	"tron-lion/internal/dao"
)

var (
	TronGenerateC = gcmd.Command{
		Name:  "gen_c",
		Usage: "tron gen_c",
		Brief: "生成数据并插入到TReceiveOrder表",
		Func:  runTronGenerateC,
	}
)

func runTronGenerateC(ctx context.Context, parser *gcmd.Parser) (err error) {
	address := g.Cfg().MustGet(ctx, "tron.address").String()
	if address == "" {
		return fmt.Errorf("未配置波场地址")
	}

	fmt.Printf("[开始] 处理初始地址: %s\n", address)

	maxDepth := g.Cfg().MustGet(ctx, "tron.maxDepth", 100).Int()
	var totalProcessed, totalInserted int
	processedAddresses := make(map[string]struct{})
	var mu sync.Mutex

	err = processAddressesWithConcurrencyCommon(
		ctx,
		[]string{address},
		0,
		maxDepth,
		processedAddresses,
		&totalProcessed,
		&totalInserted,
		&mu,
		insertOrIgnoreAddressesC,
	)
	if err != nil {
		return err
	}

	fmt.Printf("[完成] 总计处理 %d 个地址, 新增 %d 个地址\n", totalProcessed, totalInserted)
	return nil
}

func insertOrIgnoreAddressesC(ctx context.Context, addresses []string) (sql.Result, error) {
	if len(addresses) == 0 {
		return nil, nil
	}

	batch := make([]map[string]interface{}, 0, len(addresses))
	now := gtime.Now()

	for _, addr := range addresses {
		batch = append(batch, map[string]interface{}{
			dao.TReceiveOrder.Columns().OrderNo:         "ORD" + strconv.FormatInt(time.Now().UnixNano(), 10) + generateRandomString(4),
			dao.TReceiveOrder.Columns().FromAddressPart: genFromAddressPart(ctx),
			dao.TReceiveOrder.Columns().ToAddress:       addr,
			dao.TReceiveOrder.Columns().Amount:          rand.Float64()*0.008 + 0.001,
			dao.TReceiveOrder.Columns().OrderTime:       now,
			dao.TReceiveOrder.Columns().CreateTime:      now,
		})
	}

	return dao.TReceiveOrder.Ctx(ctx).
		Data(batch).
		Batch(200).
		InsertIgnore()
}
