package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/os/gtime"
)

var (
	TronPipei = gcmd.Command{
		Name:  "pipei",
		Usage: "tron pipei",
		Brief: "运行Tron前3后4地址匹配程序",
		Func:  RunTronPipei,
	}
)

func RunTronPipei(ctx context.Context, parser *gcmd.Parser) (err error) {
	// 读取配置文件中的运行时间设置（分钟）
	runMinutes := g.Cfg().MustGet(ctx, "tron.pipei.runMinutes", 5).Int() // 默认5分钟
	// 读取配置文件中的记录阈值设置
	recordThreshold := g.Cfg().MustGet(ctx, "tron.pipei.recordThreshold", 10).Int() // 默认10条
	// 读取配置文件中的GPU数量设置
	gpuCount := g.Cfg().MustGet(ctx, "tron.pipei.gpuCount", 1).Int() // 默认1个GPU
	fmt.Printf("程序将运行 %d 分钟后重启，或当新记录超过 %d 条时重启，使用 %d 个GPU\n",
		runMinutes, recordThreshold, gpuCount)

	// 不再需要用户输入GPU数量
	fmt.Println("前3后4匹配模式开始")

	// 创建一个可取消的上下文，用于控制所有任务
	rootCtx, rootCancel := context.WithCancel(ctx)
	defer rootCancel() // 确保在函数退出时取消所有任务

	// 循环执行匹配操作，直到用户手动中断
	for {
		// 记录本次循环开始时间
		loopStartTime := gtime.Now().Local()

		initialRecordCount := 0
		fmt.Printf("开始新一轮匹配，当前时间: %s\n", loopStartTime.String())

		// 设置运行截止时间
		deadline := time.Now().Add(time.Duration(runMinutes) * time.Minute)

		if err := runOnePipeiMatch(rootCtx, gpuCount, deadline, initialRecordCount, recordThreshold, loopStartTime); err != nil {
			fmt.Printf("执行匹配操作失败: %v\n", err)
		}

		fmt.Println("匹配操作完成，准备重新开始...")

		// 确保所有子任务都已终止
		fmt.Println("正在终止所有子任务...")

		// 等待一段时间确保资源被释放
		time.Sleep(5 * time.Second)

		// 强制终止所有可能仍在运行的tron.exe进程
		killTronProcesses()

		fmt.Println("所有任务已终止，准备开始新的匹配操作...")
		time.Sleep(2 * time.Second) // 短暂暂停，让用户有机会看到提示
	}
}

// 执行一次前3后4匹配操作
func runOnePipeiMatch(ctx context.Context, gpuCount int, deadline time.Time, initialRecordCount int, recordThreshold int, startTime *gtime.Time) error {
	// 获取匹配任务
	pipeiPatterns, err := getLimitedPatterns(ctx)
	if err != nil {
		return err
	}

	// 准备执行环境
	tronExePath, tempDir, gpuIdStr, err := prepareExecutionEnvironment(gpuCount)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir) // 程序结束时清理临时文件目录

	// 获取tron.exe所在目录，用于后续清理文件
	tronDir := filepath.Dir(tronExePath)

	// 先清理一次可能存在的旧临时文件
	cleanupTempFiles(tronDir)

	// 创建上下文和结果通道
	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel() // 确保函数退出时取消上下文

	// 注册信号处理，传入cancel函数
	setupSignalHandler(tempDir, cancel)

	resultChan := make(chan MatchResult, 1000)

	// 启动数据库处理goroutine
	var wgDB sync.WaitGroup
	wgDB.Add(1)
	go processResultsAndSaveToDB(ctxWithCancel, resultChan, &wgDB, ctx, pipeiPatterns)

	// 创建共用的临时文件
	tempInputFile := filepath.Join(tempDir, "input.txt")
	tempOutputFile := filepath.Join(tempDir, "output.txt")

	// 将所有匹配任务写入同一个输入文件
	var inputContent strings.Builder

	for _, pattern := range pipeiPatterns {
		inputContent.WriteString(pattern.FromAddressPart + "\n")
	}

	// 写入所有任务到临时文件
	if err := gfile.PutContents(tempInputFile, inputContent.String()); err != nil {
		return fmt.Errorf("写入临时文件失败: %v", err)
	}

	// 监控外部条件
	stopChan := make(chan struct{})
	go monitorExternalConditions(ctxWithCancel, stopChan, deadline, initialRecordCount, recordThreshold, ctx, startTime)

	// 启动文件监视器，监视共用的输出文件
	go watchOutputFile(ctxWithCancel, tempOutputFile, resultChan, 0)

	// 执行单个命令处理所有任务
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		// 构建命令
		cmd := exec.Command(tronExePath,
			"-gpu",
			"-gpuId", gpuIdStr,
			"-i", tempInputFile,
			"-o", tempOutputFile,
		)

		// 直接将命令的输出重定向到控制台
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// 输出命令信息到控制台
		fmt.Printf("执行命令: %s -gpu -gpuId %s -i %s -o %s\n",
			tronExePath, gpuIdStr, tempInputFile, tempOutputFile)

		// 执行命令
		fmt.Println("开始执行匹配任务，共有", len(pipeiPatterns), "个任务")
		if err := cmd.Run(); err != nil {
			fmt.Printf("执行命令失败: %v\n", err)
		}

		fmt.Println("匹配任务执行完成")
	}()

	// 等待任务完成或外部条件触发
	waitForCompletionOrTermination(&wg, stopChan, cancel)

	// 等待数据库处理goroutine完成
	close(resultChan)
	wgDB.Wait()

	// 清理tron.exe目录下的所有临时.txt文件
	cleanupTempFiles(tronDir)

	return nil
}
