package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/bivdex/tron-lion/internal/dao"
	"github.com/bivdex/tron-lion/internal/model/entity"

	"github.com/gogf/gf/v2/os/gcmd"
	"github.com/gogf/gf/v2/os/gtime"
)

var (
	TronGenerateC = gcmd.Command{
		Name:  "gen_c",
		Usage: "tron gen_c count prefix suffix",
		Brief: "生成订单数据并插入到TReceiveOrder表",
		Func:  runTronGenerateC,
	}
)

func runTronGenerateC(ctx context.Context, parser *gcmd.Parser) (err error) {
	// 获取参数
	count := parser.GetOpt("count").Int()
	prefix := parser.GetOpt("prefix").Int()
	suffix := parser.GetOpt("suffix").Int()
	apiAddress := parser.GetOpt("address").String()
	pageLimit := parser.GetOpt("limit").Int()
	maxPages := parser.GetOpt("pages").Int()

	// 如果没有提供参数，使用默认值
	if count <= 0 {
		count = 10 // 默认生成10条
	}
	if prefix <= 0 {
		prefix = 3 // 默认前缀长度为3
	}
	if suffix <= 0 {
		suffix = 4 // 默认后缀长度为4
	}
	if apiAddress == "" {
		apiAddress = "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf"
	}

	if pageLimit <= 0 || pageLimit > 200 {
		pageLimit = 200 // 默认每页100条，最大200条
	}
	if maxPages <= 0 {
		maxPages = 3 // 默认获取3页数据
	}

	fmt.Printf("准备生成 %d 条订单数据\n", count)
	fmt.Printf("使用API查询地址: %s (每页%d条，最多%d页)\n", apiAddress, pageLimit, maxPages)

	// 初始化随机数生成器
	rand.Seed(time.Now().UnixNano())

	// 生成并插入数据
	var orders []*entity.TReceiveOrder
	now := gtime.Now().Local()

	// 预先获取一批真实地址
	realAddresses := fetchTronAddresses(ctx, apiAddress)
	fmt.Println("获取到的真实地址: ", realAddresses)

	for i := 0; i < count; i++ {
		// 生成前缀，确保首字母是T
		prefixStr := "T" + generateRandomString(prefix-1)
		// 生成后缀
		suffixStr := generateRandomString(suffix)
		// 组合成匹配规则
		pattern := prefixStr + "*" + suffixStr

		// 生成订单号
		orderNo := "ORD" + strconv.FormatInt(time.Now().UnixNano(), 10) + strconv.Itoa(i)

		// 生成目标地址
		toAddress := getRandomAddress(realAddresses)

		// 生成随机金额 (0.001-0.005)
		amount := rand.Float64()*0.004 + 0.001

		// 生成订单时间
		orderTime := time.Now().Add(-time.Duration(rand.Intn(24)) * time.Hour).Format("2006-01-02 15:04:05")

		// 创建订单记录
		order := &entity.TReceiveOrder{
			OrderNo:         orderNo,
			FromAddressPart: pattern,
			ToAddress:       toAddress,
			Amount:          amount,
			OrderTime:       orderTime,
			CreateTime:      now,
			Initialization:  0,   // 未初始化
			ErrorData:       1,   // 正常数据
			WaitMatch:       0,   // 等待匹配
			IsDel:           "0", // 未删除
		}
		orders = append(orders, order)
	}

	// 批量插入数据库
	_, err = dao.TReceiveOrder.Ctx(ctx).Insert(orders)
	if err != nil {
		return fmt.Errorf("插入数据库失败: %v", err)
	}

	fmt.Printf("成功生成并插入 %d 条订单数据\n", count)
	// 打印部分示例
	maxDisplay := 5
	if count < maxDisplay {
		maxDisplay = count
	}
	fmt.Println("示例订单:")
	for i := 0; i < maxDisplay; i++ {
		fmt.Printf("  订单号: %s, 地址规则: %s, 金额: %.2f\n",
			orders[i].OrderNo,
			orders[i].FromAddressPart,
			orders[i].Amount)
	}
	if count > maxDisplay {
		fmt.Printf("  ... 共 %d 条\n", count)
	}

	return nil
}

// 获取一批波场真实地址
func fetchTronAddresses(ctx context.Context, apiAddress string) []string {
	// 收集的地址
	var allAddresses []string

	// 设置分页参数
	limit := 200 // 最大值200
	var fingerprint string
	maxPages := 5 // 默认5页

	for page := 0; page < maxPages; page++ {
		// 构建API URL，添加分页参数
		apiURL := fmt.Sprintf("https://api.shasta.trongrid.io/v1/accounts/%s/transactions?limit=%d",
			apiAddress, limit)

		// 如果有指纹，添加到URL
		if fingerprint != "" {
			apiURL += "&fingerprint=" + fingerprint
		}

		// 创建HTTP请求
		req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
		if err != nil {
			fmt.Printf("创建HTTP请求失败: %v\n", err)
			return generateRandomAddresses(10) // 生成10个随机地址
		}

		// 设置请求头
		req.Header.Set("Accept", "application/json")

		// 发送请求
		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("发送HTTP请求失败: %v\n", err)
			return generateRandomAddresses(10)
		}

		// 读取响应内容
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			fmt.Printf("读取响应内容失败: %v\n", err)
			return generateRandomAddresses(10)
		}

		// 解析JSON响应 - 根据实际返回的数据结构调整
		var response struct {
			Data []struct {
				RawData struct {
					Contract []struct {
						Parameter struct {
							Value struct {
								OwnerAddress string `json:"owner_address"`
								ToAddress    string `json:"to_address"`
							} `json:"value"`
						} `json:"parameter"`
					} `json:"contract"`
				} `json:"raw_data"`
			} `json:"data"`
			Success bool `json:"success"`
			Meta    struct {
				At          int64  `json:"at"`
				PageSize    int    `json:"page_size"`
				Fingerprint string `json:"fingerprint"`
			} `json:"meta"`
		}

		err = json.Unmarshal(body, &response)
		if err != nil {
			fmt.Printf("解析JSON响应失败: %v\n", err)
			return generateRandomAddresses(10)
		}

		fmt.Printf("查询结果数量: %v\n", len(response.Data))
		// 提取地址
		for _, tx := range response.Data {
			for _, contract := range tx.RawData.Contract {
				// 提取发送方地址
				// ownerAddr := hexToBase58(contract.Parameter.Value.OwnerAddress)
				// if ownerAddr != "" && !contains(allAddresses, ownerAddr) {
				// 	allAddresses = append(allAddresses, ownerAddr)
				// }

				// 提取接收方地址
				toAddr := hexToBase58(contract.Parameter.Value.ToAddress)
				if toAddr != "" && !contains(allAddresses, toAddr) {
					allAddresses = append(allAddresses, toAddr)
				}
			}
		}

		// 获取下一页的指纹
		fingerprint = response.Meta.Fingerprint
		fmt.Printf("fingerprint: %v\n", len(fingerprint))

		// 如果没有下一页，退出循环
		if fingerprint == "" {
			break
		}

		fmt.Printf("已获取第 %d 页数据，当前共有 %d 个地址\n", page+1, len(allAddresses))
	}

	// 如果没有获取到地址，生成随机地址
	if len(allAddresses) == 0 {
		fmt.Println("未从API获取到地址，使用随机生成的地址")
		return generateRandomAddresses(10)
	}

	fmt.Printf("成功从API获取到 %d 个真实地址\n", len(allAddresses))
	return allAddresses
}

// 生成随机波场地址
func generateRandomAddresses(count int) []string {
	addresses := make([]string, count)
	for i := 0; i < count; i++ {
		// 波场地址以T开头，后跟33个字符
		addresses[i] = "T" + generateRandomString(33)
	}
	fmt.Printf("已生成 %d 个随机波场地址\n", count)
	return addresses
}

// 将十六进制地址转换为Base58格式 todo
func hexToBase58(hexAddress string) string {
	if len(hexAddress) > 2 && hexAddress[:2] == "41" {
		return "T" + hexAddress[2:]
	}

	return hexAddress
}

// 检查切片中是否包含某个元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// 从地址列表中随机选择一个
func getRandomAddress(addresses []string) string {
	randomIndex := rand.Intn(len(addresses))
	return addresses[randomIndex]
}
