package generate_data

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/bivdex/tron-lion/internal/dao"
	"github.com/bivdex/tron-lion/internal/model/entity"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gtime"
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
	count := g.Cfg().MustGet(ctx, "count", 10000).Int()
	if count <= 0 {
		count = 10
	}

	fmt.Printf("准备生成 %d 条地址\n", count)

	// 初始化随机数
	rand.Seed(time.Now().UnixNano())

	// 批量生成记录
	records := make([]*entity.TOrderAddressRecordResult, 0, count)
	for i := 0; i < count; i++ {
		records = append(records, &entity.TOrderAddressRecordResult{
			FromAddressPart: genFromAddressPart(ctx),
			CreateTime:      gtime.Now(),
		})
	}

	// 批量插入
	_, err = dao.TOrderAddressRecordResult.Ctx(ctx).Insert(records)
	if err != nil {
		return fmt.Errorf("插入数据库失败: %v", err)
	}

	fmt.Printf("成功生成并插入 %d 条地址\n", count)
	return nil
}
