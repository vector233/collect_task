package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/bivdex/tron-lion/internal/model/entity"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/os/gtime"
)

var (
	TronLianghao = gcmd.Command{
		Name:  "lianghao",
		Usage: "tron lianghao",
		Brief: "运行Tron靓号匹配程序",
		Func:  RunTronLianghao,
	}
)

func RunTronLianghao(ctx context.Context, parser *gcmd.Parser) (err error) {
	// 读取配置文件中的运行时间设置（分钟）
	runMinutes := g.Cfg().MustGet(ctx, "tron.lianghao.runMinutes", 5).Int() // 默认5分钟
	// 读取配置文件中的记录阈值设置
	recordThreshold := g.Cfg().MustGet(ctx, "tron.lianghao.recordThreshold", 10).Int() // 默认10条
	// 读取配置文件中的GPU数量设置
	gpuCount := g.Cfg().MustGet(ctx, "tron.lianghao.gpuCount", 1).Int() // 默认1个GPU
	// 读取配置文件中的靓号尾数设置
	lianghao := g.Cfg().MustGet(ctx, "tron.lianghao.suffix", "").String() // 默认为空字符串

	fmt.Printf("程序将运行 %d 分钟后重启，或当新记录超过 %d 条时重启，使用 %d 个GPU\n",
		runMinutes, recordThreshold, gpuCount)
	fmt.Printf("靓号匹配模式开始，使用靓号尾数: %s\n", lianghao)

	// 如果配置文件中没有设置靓号尾数，则提示用户输入
	if lianghao == "" {
		reader := bufio.NewReader(os.Stdin)
		fmt.Print("配置文件中未设置靓号尾数，请手动输入: ")
		lianghaoInput, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		lianghao = strings.TrimSpace(lianghaoInput)
	}

	// 创建一个可取消的上下文，用于控制所有任务
	rootCtx, rootCancel := context.WithCancel(ctx)
	defer rootCancel() // 确保在函数退出时取消所有任务

	// 循环执行匹配操作，直到用户手动中断
	for {
		// 记录本次循环开始时间
		loopStartTime := gtime.Now().Local()

		// 获取初始记录数不再需要，因为我们只关心本次循环开始后的新增记录
		initialRecordCount := 0
		fmt.Printf("开始新一轮匹配，当前时间: %s\n", loopStartTime.String())

		// 设置运行截止时间
		deadline := time.Now().Add(time.Duration(runMinutes) * time.Minute)

		// 执行匹配操作，传递deadline、initialRecordCount和loopStartTime参数
		if err := runOneLianghaoMatch(rootCtx, gpuCount, lianghao, deadline, initialRecordCount, recordThreshold, loopStartTime); err != nil {
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

// 执行一次靓号匹配操作
func runOneLianghaoMatch(ctx context.Context, gpuCount int, lianghao string, deadline time.Time, initialRecordCount int, recordThreshold int, startTime *gtime.Time) error {
	// 获取匹配任务
	lianghaoPatterns, err := getLimitedPatterns(ctx)
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

	resultChan := make(chan MatchResult, 100)

	// 启动数据库处理goroutine
	var wgDB sync.WaitGroup
	wgDB.Add(1)
	go processResultsAndSaveToDB(ctxWithCancel, resultChan, &wgDB, ctx, lianghaoPatterns)

	// 创建共用的临时文件
	tempInputFile := filepath.Join(tempDir, "input.txt")

	// 将所有匹配任务写入同一个输入文件
	var inputContent strings.Builder
	patternMap := make(map[string]*entity.TOrderAddressRecordResult)

	for _, pattern := range lianghaoPatterns {
		inputContent.WriteString(pattern.FromAddressPart + "\n")
		patternMap[pattern.FromAddressPart] = pattern
	}

	// 写入所有任务到临时文件
	if err := gfile.PutContents(tempInputFile, inputContent.String()); err != nil {
		return fmt.Errorf("写入临时文件失败: %v", err)
	}

	// 监控外部条件
	stopChan := make(chan struct{})
	go monitorExternalConditions(ctxWithCancel, stopChan, deadline, initialRecordCount, recordThreshold, ctx, startTime)

	// 获取tron.exe所在目录，用于找到000.txt
	outputFile := filepath.Join(tronDir, "000.txt")

	// 启动文件监视器，监视共用的输出文件
	go watchOutputFile(ctxWithCancel, outputFile, resultChan, 0)

	// 执行单个命令处理所有任务
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		// 构建命令
		cmd := exec.Command(tronExePath,
			"-gpu",
			"-gpuId", gpuIdStr,
			"-lianghao", lianghao,
			"-i", tempInputFile,
		)

		// 直接将命令的输出重定向到控制台
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// 输出命令信息到控制台
		fmt.Printf("执行命令: %s -gpu -gpuId %s -lianghao %s -i %s\n",
			tronExePath, gpuIdStr, lianghao, tempInputFile)

		// 执行命令
		fmt.Println("开始执行匹配任务，共有", len(lianghaoPatterns), "个任务")
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

// 清理目录下的临时.txt文件
func cleanupTempFiles(dir string) {
	// 获取目录中的所有文件
	files, err := filepath.Glob(filepath.Join(dir, "*.txt"))
	if err != nil {
		fmt.Printf("获取临时文件列表失败: %v\n", err)
		return
	}

	// 保留的文件列表（不删除这些文件）
	preserveFiles := map[string]bool{
		// "lianghao.txt": true,
		// 可以添加其他需要保留的文件
	}

	// 删除临时文件
	deletedCount := 0
	for _, file := range files {
		// 获取文件名（不含路径）
		fileName := filepath.Base(file)

		// 如果是需要保留的文件，则跳过
		if preserveFiles[fileName] {
			continue
		}

		// 尝试多次删除文件，因为文件可能被其他进程占用
		for i := 0; i < 3; i++ {
			err := os.Remove(file)
			if err == nil {
				deletedCount++
				break
			} else if i == 2 {
				fmt.Printf("删除临时文件失败 %s: %v\n", file, err)
			}
			// 如果删除失败，等待一小段时间后重试
			time.Sleep(100 * time.Millisecond)
		}
	}

	fmt.Printf("已清理 %d 个临时文件\n", deletedCount)
}
