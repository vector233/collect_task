package generate_data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"sync"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/shengdoushi/base58"
)

// 处理函数类型
type AddressProcessorFunc func(ctx context.Context, addresses []string) (sql.Result, error)

// 通用并发处理函数
func processAddressesWithConcurrencyCommon(
	ctx context.Context,
	addresses []string,
	currentDepth,
	maxDepth int,
	processedAddresses map[string]struct{},
	totalProcessed,
	totalInserted *int,
	mu *sync.Mutex,
	processor AddressProcessorFunc, // 传入不同的处理函数
) error {
	if currentDepth >= maxDepth {
		fmt.Printf("[信息] 已达到最大深度 %d，停止递归\n", maxDepth)
		return nil
	}

	maxConcurrency := g.Cfg().MustGet(ctx, "tron.maxConcurrency", 10).Int()
	fmt.Printf("深度 %d/%d: 处理 %d 个地址 (并发: %d)\n",
		currentDepth+1, maxDepth, len(addresses), maxConcurrency)

	// 创建并发控制通道
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var nextLevelAddresses []string
	var nextLevelMu sync.Mutex
	var layerProcessed, layerInserted, layerSkipped int
	var layerMu sync.Mutex

	for _, addr := range addresses {
		// 检查是否已处理过该地址
		mu.Lock()
		if _, exists := processedAddresses[addr]; exists {
			mu.Unlock()
			layerMu.Lock()
			layerSkipped++
			layerMu.Unlock()
			continue
		}
		processedAddresses[addr] = struct{}{}
		mu.Unlock()

		// 并发控制
		semaphore <- struct{}{}
		wg.Add(1)

		go func(address string) {
			defer func() {
				<-semaphore
				wg.Done()
			}()

			fmt.Printf("深度 %d/%d: 处理地址: %s\n", currentDepth+1, maxDepth, address)
			addresses, err := fetchAddresses(ctx, address)
			if err != nil {
				fmt.Printf("获取地址 %s 失败: %v\n", address, err)
				return
			}

			mu.Lock()
			*totalProcessed += len(addresses)
			mu.Unlock()

			fmt.Printf("深度 %d/%d: 地址 %s 找到 %d 个相关地址\n",
				currentDepth+1, maxDepth, address, len(addresses))

			result, err := processor(ctx, addresses)
			if err != nil {
				fmt.Printf("插入地址失败: %v\n", err)
				return
			}

			insertedCount, err := result.RowsAffected()
			if err != nil {
				fmt.Printf("获取插入结果失败: %v\n", err)
				return
			}

			mu.Lock()
			*totalInserted += int(insertedCount)
			mu.Unlock()

			layerMu.Lock()
			layerInserted += int(insertedCount)
			layerProcessed += len(addresses)
			layerMu.Unlock()

			fmt.Printf("深度 %d/%d: 地址 %s 完成，新增 %d 个地址\n",
				currentDepth+1, maxDepth, address, insertedCount)

			// 将所有地址添加到下一层处理队列
			if currentDepth < maxDepth-1 {
				nextLevelMu.Lock()
				nextLevelAddresses = append(nextLevelAddresses, addresses...)
				nextLevelMu.Unlock()
			}
		}(addr)
	}

	wg.Wait()

	fmt.Printf("[统计] 深度 %d/%d: 处理 %d 个地址，跳过 %d 个，发现 %d 个，新增 %d 个\n",
		currentDepth+1, maxDepth, len(addresses), layerSkipped, layerProcessed, layerInserted)

	if currentDepth < maxDepth-1 && len(nextLevelAddresses) > 0 {
		maxAddressesPerLevel := g.Cfg().MustGet(ctx, "tron.maxAddressesPerLevel", 1000).Int()
		fmt.Printf("深度 %d/%d: 下一层有 %d 个地址，限制为 %d 个\n",
			currentDepth+1, maxDepth, len(nextLevelAddresses), maxAddressesPerLevel)

		if len(nextLevelAddresses) > maxAddressesPerLevel {
			nextLevelAddresses = nextLevelAddresses[:maxAddressesPerLevel]
		}

		return processAddressesWithConcurrencyCommon(
			ctx, nextLevelAddresses, currentDepth+1, maxDepth,
			processedAddresses, totalProcessed, totalInserted, mu, processor,
		)
	} else {
		if currentDepth >= maxDepth-1 {
			fmt.Printf("[信息] 已达到最大深度 %d\n", maxDepth)
		} else {
			fmt.Printf("[信息] 深度 %d/%d: 没有新地址，递归结束\n", currentDepth+1, maxDepth)
		}
	}

	return nil
}

// 生成随机字符串
func generateRandomString(length int) string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	result := strings.Builder{}
	for i := 0; i < length; i++ {
		result.WriteByte(charset[rand.Intn(len(charset))])
	}
	return result.String()
}

func genFromAddressPart(ctx context.Context) string {
	prefix := g.Cfg().MustGet(ctx, "tron.prefix", 3).Int()
	suffix := g.Cfg().MustGet(ctx, "tron.suffix", 4).Int()

	prefixStr := "T" + generateRandomString(prefix-1)
	suffixStr := generateRandomString(suffix)

	return prefixStr + "*" + suffixStr
}

// 十六进制地址转Base58
func hexAddressToBase58(hexAddress string) (string, error) {
	addressBytes, err := hex.DecodeString(hexAddress)
	if err != nil {
		return "", fmt.Errorf("解码地址失败: %v", err)
	}

	firstSHA := sha256.Sum256(addressBytes)
	secondSHA := sha256.Sum256(firstSHA[:])
	checksum := secondSHA[:4]

	addressWithChecksum := append(addressBytes, checksum...)
	base58Address := base58.Encode(addressWithChecksum, base58.BitcoinAlphabet)

	return base58Address, nil
}
