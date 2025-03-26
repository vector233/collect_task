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
	// 从配置中读取地址
	address := g.Cfg().MustGet(ctx, "tron.address").String()
	if address == "" {
		return fmt.Errorf("未配置波场地址，请在配置文件中设置 tron.address")
	}

	fmt.Printf("正在处理波场地址: %s\n", address)

	addresses, err := fetchAddresses(ctx, address)
	if err != nil {
		return err
	}

	fmt.Printf("找到 %d 个相关地址\n", len(addresses))

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

	fmt.Printf("已存在 %d 个地址, 需要插入 %d 个新地址\n",
		len(existingAddresses), len(newAddresses))

	// 打印部分已存在的地址作为示例
	if len(existingAddresses) > 0 {
		fmt.Println("部分已存在的地址示例:")
		i := 0
		for addr := range existingAddresses {
			if i < 10 { // 只打印前10个
				fmt.Printf("  %s\n", addr)
				i++
			} else {
				break
			}
		}
	}

	// 如果有新地址需要插入
	if len(newAddresses) > 0 {
		// 批量插入新地址
		if err := batchInsertAddresses(ctx, newAddresses); err != nil {
			return fmt.Errorf("批量插入地址失败: %v", err)
		}

		// 打印部分新插入的地址作为示例
		fmt.Println("部分新插入的地址示例:")
		for i, addr := range newAddresses {
			if i < 10 { // 只打印前10个
				fmt.Printf("  %s\n", addr)
			} else {
				break
			}
		}
	}

	fmt.Printf("处理完成: 共找到 %d 个地址, 新插入 %d 个, 已存在 %d 个\n",
		len(addresses), len(newAddresses), len(existingAddresses))

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
		Batch(100). // 每批次插入100条记录
		Insert()

	if err != nil {
		return fmt.Errorf("批量插入数据库失败: %v", err)
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
		body, err := io.ReadAll(resp.Body)
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
