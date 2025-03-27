package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"

	"github.com/bivdex/tron-lion/internal/dao"
	"github.com/bivdex/tron-lion/internal/model/entity"

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
	// 获取参数
	count := g.Cfg().MustGet(ctx, "count").Int()

	// 如果没有提供参数，使用默认值
	if count <= 0 {
		count = 10 // 默认生成10条
	}

	fmt.Printf("准备生成 %d 条地址\n", count)

	// 初始化随机数生成器
	rand.Seed(time.Now().UnixNano())

	// 生成并插入数据
	var records []*entity.TOrderAddressRecordResult
	for i := 0; i < count; i++ {
		pattern := genFromAddressPart(ctx)

		// 创建记录
		record := &entity.TOrderAddressRecordResult{
			FromAddressPart: pattern,
			CreateTime:      gtime.Now(),
		}
		records = append(records, record)
	}

	// 批量插入数据库
	_, err = dao.TOrderAddressRecordResult.Ctx(ctx).Insert(records)
	if err != nil {
		return fmt.Errorf("插入数据库失败: %v", err)
	}

	fmt.Printf("成功生成并插入 %d 条地址\n", count)
	return nil
}

// 生成指定长度的随机字符串
func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := strings.Builder{}
	for i := 0; i < length; i++ {
		randomIndex := rand.Intn(len(charset))
		result.WriteByte(charset[randomIndex])
	}
	return result.String()
}
