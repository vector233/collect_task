package cmd

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/bivdex/tron-lion/internal/dao"
	"github.com/bivdex/tron-lion/internal/model/entity"

	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gtime"
)

var (
	TronGenerate = gcmd.Command{
		Name:  "generate",
		Usage: "tron generate count prefix suffix",
		Brief: "生成前X后Y格式的地址匹配规则并插入数据库",
		Func:  runTronGenerate,
	}
)

func runTronGenerate(ctx context.Context, parser *gcmd.Parser) (err error) {
	// 获取参数
	count := parser.GetOpt("count").Int()
	prefix := parser.GetOpt("prefix").Int()
	suffix := parser.GetOpt("suffix").Int()

	// 如果没有提供参数，使用默认值
	if count <= 0 {
		count = 10 // 默认生成10条
	}
	if prefix <= 0 {
		prefix = 3 // 默认前缀长度为3
	}
	if suffix <= 0 {
		suffix = 4 // 默认后缀长度为4
	}

	fmt.Printf("准备生成 %d 条前%d后%d格式的地址匹配规则\n", count, prefix, suffix)

	// 初始化随机数生成器
	rand.Seed(time.Now().UnixNano())

	// 生成并插入数据
	var records []*entity.TOrderAddressRecordResult
	for i := 0; i < count; i++ {
		// 生成前缀，确保首字母是T
		prefixStr := "T" + generateRandomString(prefix-1)
		// 生成后缀
		suffixStr := generateRandomString(suffix)
		// 组合成匹配规则
		pattern := prefixStr + "*" + suffixStr

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

	fmt.Printf("成功生成并插入 %d 条地址匹配规则\n", count)
	// 打印部分示例
	maxDisplay := 5
	if count < maxDisplay {
		maxDisplay = count
	}
	fmt.Println("示例规则:")
	for i := 0; i < maxDisplay; i++ {
		fmt.Printf("  %s\n", records[i].FromAddressPart)
	}
	if count > maxDisplay {
		fmt.Printf("  ... 共 %d 条\n", count)
	}

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
