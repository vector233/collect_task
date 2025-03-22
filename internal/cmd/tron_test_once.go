package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bivdex/tron-lion/internal/dao"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gtime"
)

var (
	TronTest = gcmd.Command{
		Name:  "test",
		Usage: "tron test",
		Brief: "运行Tron匹配程序测试（只执行一次，不会根据配置规则重启）",
		Func:  runTronTest,
	}
)

func runTronTest(ctx context.Context, parser *gcmd.Parser) (err error) {
	// 读取配置文件中的运行时间设置（分钟）
	runMinutes := g.Cfg().MustGet(ctx, "tron.runMinutes", 5).Int() // 默认5分钟
	// 读取配置文件中的记录阈值设置
	recordThreshold := g.Cfg().MustGet(ctx, "tron.recordThreshold", 10).Int() // 默认10条
	fmt.Printf("程序将运行 %d 分钟或当新记录超过 %d 条时结束\n", runMinutes, recordThreshold)

	// 获取用户输入
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("测试匹配模式开始")
	fmt.Print("输入要跑的卡数量: ")
	input, err := reader.ReadString('\n')
	if err != nil {
		return err
	}

	// 转换输入为整数
	var gpuCount int
	_, err = fmt.Sscanf(strings.TrimSpace(input), "%d", &gpuCount)
	if err != nil {
		return fmt.Errorf("无效输入: %v", err)
	}

	// 选择匹配模式
	fmt.Println("请选择匹配模式:")
	fmt.Println("1. 匹配模式")
	fmt.Println("2. 靓号模式")
	fmt.Print("请输入选择 (1/2): ")
	modeInput, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	mode := strings.TrimSpace(modeInput)

	// 获取初始记录数
	initialRecordCount, err := dao.TOrderAddressRecordResult.Ctx(ctx).
		Where("address IS NULL OR address = ''").
		Count()
	if err != nil {
		fmt.Printf("获取初始记录数失败: %v\n", err)
		initialRecordCount = 0
	}
	fmt.Printf("初始待处理记录数: %d\n", initialRecordCount)

	// 设置运行截止时间
	deadline := time.Now().Add(time.Duration(runMinutes) * time.Minute)

	// 记录测试开始时间
	startTime := gtime.Now().Local()
	fmt.Printf("测试开始时间: %s\n", startTime.String())

	// 根据用户选择执行不同的匹配操作
	switch mode {
	case "1":
		fmt.Println("执行匹配模式测试...")
		if err := runOnePipeiMatch(ctx, gpuCount, deadline, initialRecordCount, recordThreshold, startTime); err != nil {
			fmt.Printf("执行匹配操作失败: %v\n", err)
		}
	case "2":
		fmt.Println("执行靓号模式测试...")
		// 获取靓号尾数
		fmt.Print("请输入靓号尾数: ")
		lianghaoInput, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		lianghao := strings.TrimSpace(lianghaoInput)

		if err := runOneLianghaoMatch(ctx, gpuCount, lianghao, deadline, initialRecordCount, recordThreshold, startTime); err != nil {
			fmt.Printf("执行匹配操作失败: %v\n", err)
		}
	default:
		return fmt.Errorf("无效的选择: %s", mode)
	}

	fmt.Println("测试完成")

	// 强制终止所有可能仍在运行的tron.exe进程
	killTronProcesses()

	return nil
}
