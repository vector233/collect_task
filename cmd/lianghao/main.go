package main

import (
	"context"
	"fmt"
	"os"

	"tron-lion/internal/cmd"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	_ "github.com/gogf/gf/contrib/nosql/redis/v2"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcfg"
)

func main() {
	ctx := context.Background()

	// 设置配置文件路径
	g.Cfg().GetAdapter().(*gcfg.AdapterFile).SetFileName("config.lianghao.yaml")

	fmt.Println("开始执行 Tron 靓号匹配程序...")

	// 直接调用 lianghao 命令的执行函数
	if err := cmd.RunTronLianghao(ctx, nil); err != nil {
		fmt.Printf("执行失败: %v\n", err)
		os.Exit(1)
	}
}
