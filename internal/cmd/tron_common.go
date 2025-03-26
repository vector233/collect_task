package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shengdoushi/base58"

	"github.com/bivdex/tron-lion/internal/dao"
	"github.com/bivdex/tron-lion/internal/model/entity"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/os/gtime"
)

// MatchResult 表示匹配结果
type MatchResult struct {
	Pattern string // 原始匹配模式
	Result  string // 匹配结果
}

// 准备执行环境
func prepareExecutionEnvironment(gpuCount int) (string, string, string, error) {
	// 获取当前工作目录
	currentDir, err := os.Getwd()
	if err != nil {
		return "", "", "", fmt.Errorf("获取工作目录失败: %v", err)
	}

	// 构建tron.exe的绝对路径
	tronExePath := filepath.Join(currentDir, "tron.exe")

	// 检查文件是否存在
	if !gfile.Exists(tronExePath) {
		return "", "", "", fmt.Errorf("未找到tron.exe: %s", tronExePath)
	}

	// 构建 gpuId 参数
	var gpuIds []string
	for i := 0; i < gpuCount; i++ {
		gpuIds = append(gpuIds, fmt.Sprintf("%d", i))
	}
	gpuIdStr := strings.Join(gpuIds, ",")

	// 创建临时文件目录
	tempDir := filepath.Join(currentDir, "temp_files")

	// 先检查并清理可能存在的旧临时文件目录
	if gfile.Exists(tempDir) {
		fmt.Println("发现旧的临时文件目录，正在清理...")
		if err := os.RemoveAll(tempDir); err != nil {
			fmt.Printf("清理旧临时文件目录失败: %v\n", err)
		}
	}

	// 创建新的临时文件目录
	fmt.Printf("创建临时文件目录: %s\n", tempDir)
	if err := os.MkdirAll(tempDir, 0666); err != nil {
		return "", "", "", fmt.Errorf("创建临时文件目录失败: %v", err)
	}

	// 验证目录是否创建成功
	if !gfile.Exists(tempDir) {
		return "", "", "", fmt.Errorf("临时文件目录创建失败，路径不存在: %s", tempDir)
	}

	fmt.Printf("临时文件目录创建成功: %s\n", tempDir)
	return tronExePath, tempDir, gpuIdStr, nil
}

// 设置信号处理
func setupSignalHandler(tempDir string, cancel context.CancelFunc) chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("接收到中断信号，正在清理临时文件...")
		// 先取消上下文，通知所有goroutine停止
		cancel()
		// 等待一小段时间让goroutine有机会清理
		time.Sleep(500 * time.Millisecond)
		// 清理临时文件
		os.RemoveAll(tempDir)

		// 清理tron.exe所在目录下的所有临时.txt文件
		currentDir, err := os.Getwd()
		if err == nil {
			cleanupTempFiles(currentDir)
		}

		// 强制终止所有可能仍在运行的tron.exe进程
		killTronProcesses()

		// 强制退出程序
		os.Exit(1)
	}()
	return sigChan
}

// 处理结果并保存到数据库
func processResultsAndSaveToDB(ctx context.Context, resultChan <-chan MatchResult, wg *sync.WaitGroup, dbCtx context.Context, patterns []*entity.TOrderAddressRecordResult) {
	defer wg.Done()

	// 批处理大小和计时器，用于定期将结果写入数据库
	const batchSize = 10
	var resultBatch []MatchResult
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// 创建任务map，用于正则判断结果是否属于该任务
	patternMap := make(map[string]*entity.TOrderAddressRecordResult)
	for _, p := range patterns {
		patternMap[p.FromAddressPart] = p
	}

	// 保存结果到数据库的函数
	saveResultsToDB := func() {
		if len(resultBatch) > 0 {
			for _, matchResult := range resultBatch {
				// 解析地址和私钥
				parts := strings.Split(matchResult.Result, "---")
				address := matchResult.Pattern // 地址
				privateAddress := ""
				if len(parts) > 1 {
					privateAddress = parts[1]
				}

				// 使用本地时区的当前时间
				now := gtime.Now().Local()

				// 匹配到的任务
				var matchedPattern string
				var matchedRecord *entity.TOrderAddressRecordResult

				// 遍历所有任务，查找匹配的模式
				for pattern, record := range patternMap {
					// 检查任务是否匹配地址
					if matchesPattern(address, pattern) {
						matchedPattern = pattern
						matchedRecord = record
						break
					}
				}

				// 如果找不到匹配的任务，则忽略该结果
				if matchedPattern == "" || matchedRecord == nil {
					// fmt.Printf("找不到匹配的任务模式，忽略结果: Address=%s\n", address)
					continue
				}

				// 如果记录已存在且address为空，则更新记录
				if matchedRecord.Address == "" {
					// 更新记录
					_, err := dao.TOrderAddressRecordResult.Ctx(dbCtx).
						Data(g.Map{
							"address":            address,
							"private_address":    privateAddress,
							"match_success_time": now,
						}).
						Where("from_address_part = ?", matchedPattern).
						Update()

					if err != nil {
						fmt.Printf("更新结果到数据库失败: %v\n", err)
					}
					//} else {
					//fmt.Printf("任务记录已有结果，忽略更新: FromAddressPart=%s\n", matchedPattern)
				}
			}

		}
	}

	for {
		select {
		case <-ctx.Done():
			// 上下文被取消，保存剩余结果并退出
			saveResultsToDB()
			return
		case matchResult, ok := <-resultChan:
			if !ok {
				// 通道已关闭，保存剩余结果并退出
				saveResultsToDB()
				return
			}

			// 添加到批处理
			resultBatch = append(resultBatch, matchResult)

			// 当批处理达到一定大小时，保存到数据库
			if len(resultBatch) >= batchSize {
				saveResultsToDB()
			}
		case <-ticker.C:
			// 定期保存结果到数据库，即使批处理未满
			saveResultsToDB()
		}
	}
}

// 检查地址是否匹配任务
func matchesPattern(address, pattern string) bool {
	// 如果模式中不包含通配符，则直接比较
	if !strings.Contains(pattern, "*") {
		return address == pattern
	}

	// 将任务转换为正则表达式
	regexPattern := strings.Replace(pattern, "*", ".*", -1)
	regexPattern = "^" + regexPattern + "$"

	matched, err := regexp.MatchString(regexPattern, address)
	if err != nil {
		fmt.Printf("正则匹配错误: %v\n", err)
		return false
	}

	return matched
}

// 监控外部条件
func monitorExternalConditions(ctx context.Context, stopChan chan<- struct{}, deadline time.Time, initialRecordCount, recordThreshold int, dbCtx context.Context, startTime *gtime.Time) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(10 * time.Second): // 每10秒检查一次
			// 检查是否已经到达截止时间
			if time.Now().After(deadline) {
				fmt.Println("已达到设定的运行时间，准备终止程序...")
				close(stopChan)
				return
			}

			// 检查任务数是否超过阈值（只统计本轮开始后新增的待计算任务）
			newRecords, err := dao.TOrderAddressRecordResult.Ctx(dbCtx).
				Where("(address IS NULL OR address = '') AND create_time > ?", startTime).
				Count()
			if err != nil {
				fmt.Printf("获取当前任务数失败: %v\n", err)
			} else {
				//newRecords := currentRecordCount
				//fmt.Printf("本轮开始后新增待匹配记录数: %d, 阈值: %d\n",
				//	newRecords, recordThreshold)

				// 如果新增记录数超过阈值，则准备终止任务
				if newRecords >= recordThreshold {
					fmt.Printf("新增待匹配任务数 %d 超过阈值 %d，准备终止任务...\n", newRecords, recordThreshold)
					close(stopChan)
					return
				}
			}
		}
	}
}

// 等待任务完成或终止
func waitForCompletionOrTermination(wg *sync.WaitGroup, stopChan <-chan struct{}, cancelFunc context.CancelFunc) {
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		fmt.Println("所有任务已完成")
	case <-stopChan:
		fmt.Println("由于外部条件触发，正在终止任务...")
		cancelFunc() // 取消上下文，通知所有goroutine停止
	}
}

// 监视输出文件
func watchOutputFile(ctx context.Context, outputFile string, resultChan chan<- MatchResult, idx int) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	failedFile := "failed_private_address.log"
	mismatchedFile := "mismatched_private_address.log"
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// 检查文件是否存在
			if !gfile.Exists(outputFile) {
				continue
			}

			// 创建备份文件名
			oldFile := outputFile + ".old"

			// 先读取原文件内容
			content := gfile.GetContents(outputFile)

			// 如果文件为空，则继续
			if content == "" {
				continue
			}

			// 原文件重命名为 xxx.old
			err := os.Rename(outputFile, oldFile)
			if err != nil {
				fmt.Printf("重命名原文件失败 [%d]: %v\n", idx, err)
				continue
			}

			// 直接创建一个新的空文件（使用原文件名）
			err = gfile.PutContents(outputFile, "")
			if err != nil {
				fmt.Printf("创建新文件失败 [%d]: %v\n", idx, err)
				// 如果创建新文件失败，尝试恢复原文件
				os.Rename(oldFile, outputFile)
				continue
			}

			// 将内容分割为行
			lines := strings.Split(content, "\n")

			// 处理所有行
			for _, line := range lines {
				if line == "" {
					continue // 跳过空行
				}

				// 解析行内容，格式应该是 "地址---私钥"
				parts := strings.SplitN(line, "---", 2)
				if len(parts) >= 2 {
					// 获取地址部分作为匹配模式
					addressPart := parts[0]
					privateAddress := parts[1]
					address, err := getAddressFromPrivateKey(privateAddress)
					if err != nil {
						// 写入失败的私钥到文件
						gfile.PutContentsAppend(failedFile, line+"\n")
						fmt.Printf("解析私钥失败，地址：%v， err: %v\n", line, err)
						continue
					}
					if address != addressPart {
						// 写入失败的私钥到文件
						gfile.PutContentsAppend(mismatchedFile, line+"\n")
						fmt.Printf("地址不匹配，地址：%v， err: %v\n", line, err)
						continue
					}

					// 创建匹配结果结构体
					matchResult := MatchResult{
						Pattern: addressPart, // 使用地址部分作为模式
						Result:  line,        // 保存完整结果
					}

					// 尝试发送到处理通道
					select {
					case resultChan <- matchResult:
						fmt.Printf("发送结果到处理通道: %s\n", addressPart)
					case <-ctx.Done():
						// 删除临时文件
						os.Remove(oldFile)
						return
					}
				} else {
					fmt.Printf("无法解析结果行: %s\n", line)
				}
			}

			// 处理完成后删除旧文件
			os.Remove(oldFile)
		}
	}
}

// 终止所有tron.exe进程
func killTronProcesses() {
	cmd := exec.Command("taskkill", "/F", "/IM", "tron.exe")
	err := cmd.Run()
	if err != nil {
		fmt.Printf("终止tron.exe进程失败: %v\n", err)
	} else {
		fmt.Println("已终止所有tron.exe进程")
	}
}

// 获取待执行任务
func getPatterns(ctx context.Context) ([]*entity.TOrderAddressRecordResult, error) {
	var patterns []*entity.TOrderAddressRecordResult
	err := dao.TOrderAddressRecordResult.Ctx(ctx).
		Where("address IS NULL OR address = ''").
		Scan(&patterns)
	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %v", err)
	}
	return patterns, nil
}

// 获取限制数量的待执行任务
func getLimitedPatterns(ctx context.Context) ([]*entity.TOrderAddressRecordResult, error) {
	var patterns []*entity.TOrderAddressRecordResult

	limit := g.Cfg().MustGet(ctx, "tron.pipei.limit", 30000).Int() // 默认30000条
	// 如果限制为0或负数，则使用原有的不限制查询
	if limit <= 0 {
		return getPatterns(ctx)
	}

	err := dao.TOrderAddressRecordResult.Ctx(ctx).
		Where("address IS NULL OR address = ''").
		Limit(limit).
		Scan(&patterns)
	if err != nil {
		return nil, fmt.Errorf("获取任务失败: %v", err)
	}

	fmt.Printf("已获取 %d/%d 条待执行任务\n", len(patterns), limit)
	return patterns, nil
}

// 从私钥获取波场地址
func getAddressFromPrivateKey(privateKeyHex string) (string, error) {
	// 1. 解码私钥
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return "", fmt.Errorf("解码私钥失败: %v", err)
	}

	// 2. 从私钥生成公钥
	privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
	publicKey := privateKey.PubKey()
	publicKeyBytes := publicKey.SerializeUncompressed()

	// 3. 只保留X和Y坐标，去掉前缀0x04
	publicKeyBytes = publicKeyBytes[1:]

	// 4. 对公钥进行Keccak-256哈希
	publicKeyHash := crypto.Keccak256(publicKeyBytes)

	// 5. 只保留哈希的最后20字节作为地址
	address := publicKeyHash[len(publicKeyHash)-20:]

	// 6. 添加前缀0x41（波场地址前缀）
	addressWithPrefix := append([]byte{0x41}, address...)

	// 7. 计算校验和（两次SHA-256哈希的前4字节）
	firstSHA := sha256.Sum256(addressWithPrefix)
	secondSHA := sha256.Sum256(firstSHA[:])
	checksum := secondSHA[:4]

	// 8. 将地址和校验和拼接
	addressWithChecksum := append(addressWithPrefix, checksum...)

	// 9. Base58编码得到最终地址
	tronAddress := base58.Encode(addressWithChecksum, base58.BitcoinAlphabet)

	return tronAddress, nil
}

// 将十六进制格式的波场地址转换为Base58格式
func hexAddressToBase58(hexAddress string) (string, error) {
	// 1. 解码十六进制地址
	addressBytes, err := hex.DecodeString(hexAddress)
	if err != nil {
		return "", fmt.Errorf("解码地址失败: %v", err)
	}

	// 2. 计算校验和（两次SHA-256哈希的前4字节）
	firstSHA := sha256.Sum256(addressBytes)
	secondSHA := sha256.Sum256(firstSHA[:])
	checksum := secondSHA[:4]

	// 3. 将地址和校验和拼接
	addressWithChecksum := append(addressBytes, checksum...)

	// 4. Base58编码得到最终地址
	base58Address := base58.Encode(addressWithChecksum, base58.BitcoinAlphabet)

	return base58Address, nil
}
