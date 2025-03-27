package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gtime"

	"github.com/bivdex/tron-lion/internal/dao"
)

var (
	TronGenerateE = gcmd.Command{
		Name:  "gen_e",
		Usage: "tron gen_e",
		Brief: "生成数据并插入到TOrderFromAddress表",
		Func:  runTronGenerateE,
	}
)

func runTronGenerateE(ctx context.Context, parser *gcmd.Parser) (err error) {
	// 从配置读取初始地址
	address := g.Cfg().MustGet(ctx, "tron.address").String()
	if address == "" {
		return fmt.Errorf("未配置波场地址")
	}

	fmt.Printf("[开始] 处理初始地址: %s\n", address)

	// 递归深度限制，避免无限递归
	maxDepth := g.Cfg().MustGet(ctx, "tron.maxDepth", 3).Int()

	// 记录总共处理的地址数量
	var totalProcessed, totalInserted int

	// 记录已处理地址，防止重复处理
	processedAddresses := make(map[string]struct{})

	// 开始递归处理
	err = processAddressRecursively(ctx, address, 0, maxDepth, processedAddresses, &totalProcessed, &totalInserted)
	if err != nil {
		return err
	}

	fmt.Printf("[完成] 总计处理 %d 个地址, 新增 %d 个地址\n", totalProcessed, totalInserted)
	return nil
}

// 递归处理地址及其交易记录中的地址
func processAddressRecursively(
	ctx context.Context,
	address string,
	currentDepth,
	maxDepth int,
	processedAddresses map[string]struct{},
	totalProcessed,
	totalInserted *int,
) error {
	// 检查是否已处理过该地址
	if _, exists := processedAddresses[address]; exists {
		return nil
	}

	// 标记该地址为已处理
	processedAddresses[address] = struct{}{}

	// 获取该地址的交易记录中的所有地址
	fmt.Printf("深度 %d/%d: 正在处理地址: %s\n", currentDepth+1, maxDepth, address)
	addresses, err := fetchAddresses(ctx, address)
	if err != nil {
		return fmt.Errorf("获取地址 %s 的交易记录失败: %v", address, err)
	}

	*totalProcessed += len(addresses)
	fmt.Printf("深度 %d/%d: 地址 %s 找到 %d 个相关地址\n",
		currentDepth+1, maxDepth, address, len(addresses))

	// 批量查询数据库中已存在的地址
	existingAddresses, err := batchCheckAddresses(ctx, addresses)
	if err != nil {
		return fmt.Errorf("批量查询地址失败: %v", err)
	}

	// 找出需要插入的新地址
	var newAddresses []string
	for _, addr := range addresses {
		if _, exists := existingAddresses[addr]; !exists {
			newAddresses = append(newAddresses, addr)
		}
	}

	fmt.Printf("深度 %d/%d: 已存在 %d 个地址, 需要插入 %d 个新地址\n",
		currentDepth+1, maxDepth, len(existingAddresses), len(newAddresses))

	// 如果有新地址需要插入
	if len(newAddresses) > 0 {
		// 批量插入新地址
		if err := batchInsertAddresses(ctx, newAddresses); err != nil {
			return fmt.Errorf("批量插入地址失败: %v", err)
		}

		*totalInserted += len(newAddresses)

		// 打印部分新插入的地址作为示例
		if len(newAddresses) > 0 {
			fmt.Printf("深度 %d/%d: 部分新插入的地址示例:\n", currentDepth+1, maxDepth)
			for i, addr := range newAddresses {
				if i < 5 { // 只打印前5个
					fmt.Printf("  %s\n", addr)
				} else {
					break
				}
			}
		}
	}

	// 如果未达到最大深度，继续递归处理新地址
	if currentDepth < maxDepth-1 && len(newAddresses) > 0 {
		// 限制每层递归处理的地址数量，避免爆炸式增长
		maxAddressesPerLevel := g.Cfg().MustGet(ctx, "tron.maxAddressesPerLevel", 10).Int()
		fmt.Printf("限制每层递归处理的地址数量为 %d 个\n", maxAddressesPerLevel)
		processCount := len(newAddresses)
		if processCount > maxAddressesPerLevel {
			processCount = maxAddressesPerLevel
			fmt.Printf("深度 %d/%d: 限制处理地址数量为 %d 个\n",
				currentDepth+2, maxDepth, processCount)
		}

		for i := 0; i < processCount; i++ {
			fmt.Printf("深度 %d/%d:  总共 %d 个，当前处理第 %d 个\n",
				currentDepth+2, maxDepth, processCount, i)
			err := processAddressRecursively(
				ctx,
				newAddresses[i],
				currentDepth+1,
				maxDepth,
				processedAddresses,
				totalProcessed,
				totalInserted,
			)
			if err != nil {
				fmt.Printf("处理地址 %s 时出错: %v\n", newAddresses[i], err)
				// 继续处理其他地址，不中断整个流程
				continue
			}
		}
	}

	return nil
}

// 批量检查地址是否存在于数据库
func batchCheckAddresses(ctx context.Context, addresses []string) (map[string]struct{}, error) {
	if len(addresses) == 0 {
		return make(map[string]struct{}), nil
	}

	// 查询数据库中已存在的地址
	records, err := dao.TOrderFromAddress.Ctx(ctx).
		Where(dao.TOrderFromAddress.Columns().FromAddress, addresses).
		Fields(dao.TOrderFromAddress.Columns().FromAddress).
		All()

	if err != nil {
		return nil, fmt.Errorf("查询数据库失败: %v", err)
	}

	// 将查询结果转换为map便于快速查找
	existingAddresses := make(map[string]struct{}, len(records))
	for _, record := range records {
		addr := record["from_address"].String()
		existingAddresses[addr] = struct{}{}
	}

	return existingAddresses, nil
}

// 批量插入地址到数据库
func batchInsertAddresses(ctx context.Context, addresses []string) error {
	if len(addresses) == 0 {
		return nil
	}

	// 准备批量插入的数据
	batch := make([]map[string]interface{}, 0, len(addresses))
	now := gtime.Now()

	for _, addr := range addresses {
		batch = append(batch, map[string]interface{}{
			dao.TOrderFromAddress.Columns().FromAddress: addr,
			dao.TOrderFromAddress.Columns().CreateTime:  now,
		})
	}

	// 执行批量插入
	_, err := dao.TOrderFromAddress.Ctx(ctx).
		Data(batch).
		Batch(200).
		Insert()

	if err != nil {
		return fmt.Errorf("批量插入数据库失败: %v", err)
	}

	return nil
}

// 根据波场地址查询相关交易地址
// API文档: https://developers.tron.network/reference/get-transaction-info-by-account-address
func fetchAddresses(ctx context.Context, address string) ([]string, error) {
	baseURL := "https://api.shasta.trongrid.io/v1/accounts/%s/transactions"
	apiURL := fmt.Sprintf(baseURL, address)

	// 查询参数设置
	params := url.Values{}
	params.Add("limit", "200") // 单页最大记录数
	// params.Add("only_confirmed", "true")

	// 存储发现的地址
	addressMap := make(map[string]struct{})
	addressMap[address] = struct{}{} // 包含查询地址本身

	fingerprint := ""
	hasMore := true

	// 分页查询限制，防止单地址消耗过多资源
	maxPages := g.Cfg().MustGet(ctx, "tron.maxPagesPerAddress", 10).Int()
	currentPage := 0

	// 分页获取交易记录
	for hasMore && (maxPages <= 0 || currentPage < maxPages) {
		currentPage++
		fmt.Printf("正在查询地址 %s 的第 %d/%d 页交易记录\n", address, currentPage, maxPages)

		// 构建请求URL
		fullURL := apiURL
		if len(params) > 0 {
			fullURL = fmt.Sprintf("%s?%s", apiURL, params.Encode())
		}

		// 发送请求
		req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %v", err)
		}

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("请求失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API返回错误: %d", resp.StatusCode)
		}

		// 解析响应
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("读取响应失败: %v", err)
		}

		var response TronTransactionResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("解析JSON失败: %v", err)
		}
		fmt.Printf("获取到 %d 条交易记录\n", len(response.Data))

		// 提取交易中的地址
		for _, tx := range response.Data {
			// 合约可能为空，需检查
			if len(tx.RawData.Contract) > 0 {
				// 发送方地址
				if tx.RawData.Contract[0].Parameter.Value.OwnerAddress != "" {
					hexAddress := tx.RawData.Contract[0].Parameter.Value.OwnerAddress
					addressMap[hexAddress] = struct{}{}
				}

				// 接收方地址
				if tx.RawData.Contract[0].Parameter.Value.ToAddress != "" {
					hexAddress := tx.RawData.Contract[0].Parameter.Value.ToAddress
					addressMap[hexAddress] = struct{}{}
				}
			}

			for _, internalTx := range tx.InternalTransactions {
				if internalTx.FromAddress != "" {
					addressMap[internalTx.FromAddress] = struct{}{}
				}
				if internalTx.ToAddress != "" {
					addressMap[internalTx.ToAddress] = struct{}{}
				}
			}
		}

		// 检查是否有更多页
		if response.Meta.Fingerprint != "" && len(response.Data) > 0 {
			fingerprint = response.Meta.Fingerprint
			params.Set("fingerprint", fingerprint)
		} else {
			hasMore = false
		}

		// 防止API限流，加点延迟
		time.Sleep(200 * time.Millisecond)
	}

	// 达到页数限制提示
	if hasMore && maxPages > 0 && currentPage >= maxPages {
		fmt.Printf("地址 %s 的交易记录超过最大查询页数 %d，停止查询\n", address, maxPages)
	}

	// 将map转换为slice
	addresses := make([]string, 0, len(addressMap))
	for addr := range addressMap {
		// 如果地址是十六进制格式，转换为Base58格式
		if strings.HasPrefix(addr, "41") {
			base58Addr, err := hexAddressToBase58(addr)
			if err == nil {
				addresses = append(addresses, base58Addr)
			} else {
				// 如果转换失败，仍然添加原始地址
				fmt.Printf("转换地址 %s 为Base58格式时出错: %v\n", addr, err)
				// addresses = append(addresses, addr)
			}
		} else {
			addresses = append(addresses, addr)
		}
	}

	return addresses, nil
}

// TronTransactionResponse 波场交易API的响应结构
type TronTransactionResponse struct {
	Data []struct {
		TxID                 string                `json:"txID"`
		BlockNumber          int64                 `json:"blockNumber"`
		BlockTimestamp       int64                 `json:"block_timestamp"`
		ContractResult       []string              `json:"contractResult"`
		ContractType         string                `json:"contract_type"`
		Fee                  int64                 `json:"fee"`
		RawData              RawData               `json:"raw_data"`
		InternalTransactions []InternalTransaction `json:"internal_transactions"`
	} `json:"data"`
	Success bool `json:"success"`
	Meta    struct {
		At          int64  `json:"at"`
		Fingerprint string `json:"fingerprint"`
	} `json:"meta"`
}

// RawData 交易数据
type RawData struct {
	Contract []struct {
		Parameter struct {
			Value struct {
				Amount       int64  `json:"amount"`
				OwnerAddress string `json:"owner_address"`
				ToAddress    string `json:"to_address"`
			} `json:"value"`
			TypeURL string `json:"type_url"`
		} `json:"parameter"`
		Type string `json:"type"`
	} `json:"contract"`
	RefBlockBytes string `json:"ref_block_bytes"`
	RefBlockHash  string `json:"ref_block_hash"`
	Expiration    int64  `json:"expiration"`
	Timestamp     int64  `json:"timestamp"`
}

type InternalTransaction struct {
	FromAddress string `json:"from_address"`
	ToAddress   string `json:"to_address"`
	CallValue   int64  `json:"call_value"`
}
