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
)

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
		//time.Sleep(200 * time.Millisecond)
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
