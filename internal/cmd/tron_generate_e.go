package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gogf/gf/v2/os/gcmd"
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
	// 示例调用
	address := "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf" // 替换为您要查询的地址
	addresses, err := fetchAddresses(ctx, address)
	if err != nil {
		return err
	}

	fmt.Printf("找到 %d 个相关地址\n", len(addresses))
	for i, addr := range addresses {
		if i < 10 { // 只打印前10个作为示例
			fmt.Println(addr)
		}
	}

	return nil
}

// 根据某个 波场地址查询该地址所有的交易记录下的地址
// https://developers.tron.network/reference/get-transaction-info-by-account-address
func fetchAddresses(ctx context.Context, address string) ([]string, error) {
	baseURL := "https://api.shasta.trongrid.io/v1/accounts/%s/transactions"
	apiURL := fmt.Sprintf(baseURL, address)

	// 设置查询参数
	params := url.Values{}
	params.Add("limit", "200") // 使用最大限制
	// params.Add("only_confirmed", "true") // 只获取已确认的交易

	// 用于存储所有找到的地址
	addressMap := make(map[string]struct{})

	// 添加查询地址本身
	addressMap[address] = struct{}{}

	fingerprint := ""
	hasMore := true

	// 使用分页获取所有交易
	for hasMore {
		// 构建完整URL
		fullURL := apiURL
		if len(params) > 0 {
			fullURL = fmt.Sprintf("%s?%s", apiURL, params.Encode())
		}

		// 发送HTTP请求
		req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		if err != nil {
			return nil, fmt.Errorf("创建请求失败: %v", err)
		}

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("发送请求失败: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
		}

		// 读取响应内容
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("读取响应失败: %v", err)
		}

		// 解析JSON响应
		var response TronTransactionResponse
		if err := json.Unmarshal(body, &response); err != nil {
			return nil, fmt.Errorf("解析JSON失败: %v", err)
		}

		// 提取交易中的地址
		for _, tx := range response.Data {
			// 添加from地址
			if tx.RawData.Contract[0].Parameter.Value.OwnerAddress != "" {
				hexAddress := tx.RawData.Contract[0].Parameter.Value.OwnerAddress
				addressMap[hexAddress] = struct{}{}
			}

			// 添加to地址
			if tx.RawData.Contract[0].Parameter.Value.ToAddress != "" {
				hexAddress := tx.RawData.Contract[0].Parameter.Value.ToAddress
				addressMap[hexAddress] = struct{}{}
			}

			// 处理内部交易的地址
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

		// 避免API限流
		time.Sleep(200 * time.Millisecond)
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
