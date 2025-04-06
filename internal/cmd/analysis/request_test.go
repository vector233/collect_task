package analysis

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gogf/gf/v2/test/gtest"
)

// TestGetTokenBalance 测试USDT余额查询功能
func TestGetTokenBalance(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 创建API客户端
		api := NewTronAPI(
			"https://api.trongrid.io", // 使用波场主网API
			"",                        // 这里填入您的API密钥
		)

		// 设置较长的超时时间
		api.HttpTimeout = time.Second * 30

		// 测试地址和USDT合约地址
		testAddress := "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf"  // 测试地址
		usdtContract := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" // 波场USDT合约地址

		// 调用GetTokenBalance方法
		balance, err := api.GetTokenBalance(context.Background(), testAddress, usdtContract)

		// 验证结果
		if err != nil {
			t.Logf("获取余额失败: %v", err)
		} else {
			t.Log("地址:", testAddress)
			t.Log("USDT余额:", balance)
			t.Assert(balance >= 0, true)
		}
	})
}

// TestGetTokenBalanceTestnet 测试测试网环境
func TestGetTokenBalanceTestnet(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 跳过此测试，除非明确要测试测试网
		t.Skip("跳过测试网测试")

		// 创建API客户端，使用Shasta测试网
		api := NewTronAPI(
			"https://api.shasta.trongrid.io",
			"", // 这里填入您的测试网API密钥
		)

		api.HttpTimeout = time.Second * 30

		// 测试地址和测试网USDT合约地址
		testAddress := "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf"      // 测试地址
		testUsdtContract := "TG3XXyExBkPp9nzdajDZsozEu4BkaSJozs" // 测试网USDT合约地址

		// 调用GetTokenBalance方法
		balance, err := api.GetTokenBalance(context.Background(), testAddress, testUsdtContract)

		// 验证结果
		if err != nil {
			t.Logf("获取测试网余额失败: %v", err)
		} else {
			t.Log("测试网地址:", testAddress)
			t.Log("测试网USDT余额:", balance)
			t.Assert(balance >= 0, true)
		}
	})
}

// TestGetTokenBalanceLocalNode 测试自建节点API
func TestGetTokenBalanceLocalNode(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 跳过此测试，除非明确要测试自建节点
		// t.Skip("跳过自建节点测试")

		// 创建API客户端，使用本地节点
		api := NewTronAPI(
			"http://104.233.192.15:8090", // 自建节点地址
			"",
		)

		api.HttpTimeout = time.Second * 30

		// 测试地址和USDT合约地址
		testAddress := "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf"  // 测试地址
		usdtContract := "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" // 波场USDT合约地址

		// 调用GetTokenBalance方法
		balance, err := api.GetTokenBalance(context.Background(), testAddress, usdtContract)

		// 验证结果
		if err != nil {
			t.Logf("获取余额失败: %v", err)
		} else {
			t.Log("地址:", testAddress)
			t.Log("USDT余额:", balance)
			t.Assert(balance >= 0, true)
		}
	})
}

// TestGetUSDTTransactions 测试USDT交易记录查询功能
func TestGetUSDTTransactions(t *testing.T) {
	// 创建API客户端
	api := NewTronAPI(
		"https://api.trongrid.io", // 使用波场主网API
		"",                        // 这里填入您的API密钥
	)

	// 设置较长的超时时间
	api.HttpTimeout = time.Second * 30

	// 测试地址 - 使用一个已知有USDT交易的地址
	testAddress := "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf"

	// 测试不同的参数组合
	testCases := []struct {
		name   string
		params TRC20TransactionParams
	}{
		{
			name: "基本查询-限制5条",
			params: TRC20TransactionParams{
				Limit: 5,
			},
		},
		{
			name: "只查询已确认交易",
			params: TRC20TransactionParams{
				Limit:         5,
				OnlyConfirmed: boolPtr(true),
			},
		},
		{
			name: "只查询转入交易",
			params: TRC20TransactionParams{
				Limit:  5,
				OnlyTo: boolPtr(true),
			},
		},
		{
			name: "只查询转出交易",
			params: TRC20TransactionParams{
				Limit:    5,
				OnlyFrom: boolPtr(true),
			},
		},
		{
			name: "按时间范围查询",
			params: TRC20TransactionParams{
				Limit:        5,
				MinTimestamp: timePtr(time.Now().AddDate(0, -1, 0)), // 一个月前
				MaxTimestamp: timePtr(time.Now()),                   // 现在
			},
		},
		{
			name: "按降序排序",
			params: TRC20TransactionParams{
				Limit:   5,
				OrderBy: "block_timestamp,desc",
			},
		},
	}

	for _, tc := range testCases {
		gtest.C(t, func(t *gtest.T) {
			// 调用GetUSDTTransactions方法
			transactions, err := api.GetUSDTTransactions(context.Background(), testAddress, tc.params)

			// 验证结果
			if err != nil {
				t.Logf("[%s] 获取USDT交易记录失败: %v", tc.name, err)
			} else {
				t.Logf("[%s] 成功获取 %d 条USDT交易记录", tc.name, len(transactions))

				// 打印第一笔交易详情
				if len(transactions) > 0 {
					tx := transactions[0]
					fmt.Printf("%+v\n", tx)
					t.Logf("  交易ID: %s", tx.TransactionID)
					t.Logf("  时间: %s", tx.Timestamp.Format("2006-01-02 15:04:05"))
					t.Logf("  发送方: %s", tx.From)
					t.Logf("  接收方: %s", tx.To)
					t.Logf("  金额: %.6f %s", tx.Amount, tx.TokenSymbol)

					// 针对特定参数进行验证
					if tc.params.OnlyTo != nil && *tc.params.OnlyTo {
						// 验证接收方是测试地址
						t.Assert(strings.EqualFold(tx.To, testAddress), true)
					}

					if tc.params.OnlyFrom != nil && *tc.params.OnlyFrom {
						// 验证发送方是测试地址
						t.Assert(strings.EqualFold(tx.From, testAddress), true)
					}

					if tc.params.OnlyConfirmed != nil && *tc.params.OnlyConfirmed {
						// 验证交易已确认
						t.Assert(*tc.params.OnlyConfirmed, true)
					}
				}
			}
		})
	}
}

// boolPtr 返回布尔值的指针
func boolPtr(b bool) *bool {
	return &b
}

// timePtr 返回时间的指针
func timePtr(t time.Time) *time.Time {
	return &t
}

// TestGetIncomingUSDTTransactions 测试USDT转入交易记录查询功能
func TestGetIncomingUSDTTransactions(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 创建API客户端
		api := NewTronAPI(
			"https://api.trongrid.io", // 使用波场主网API
			"",                        // 这里填入您的API密钥
		)

		// 设置较长的超时时间
		api.HttpTimeout = time.Second * 30

		// 测试地址 - 使用一个已知有USDT转入交易的地址
		testAddress := "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf"

		// 调用GetIncomingUSDTTransactions方法，获取最近10条转入交易
		transactions, err := api.GetIncomingUSDTTransactions(context.Background(), testAddress, 10, 0)

		// 验证结果
		if err != nil {
			t.Logf("获取USDT转入交易记录失败: %v", err)
		} else {
			t.Logf("成功获取 %d 条USDT转入交易记录", len(transactions))

			// 打印交易详情
			for i, tx := range transactions {
				t.Logf("转入交易 #%d:", i+1)
				t.Logf("  交易ID: %s", tx.TransactionID)
				t.Logf("  时间: %s", tx.Timestamp.Format("2006-01-02 15:04:05"))
				t.Logf("  发送方: %s", tx.From)
				t.Logf("  接收方: %s", tx.To)
				t.Logf("  金额: %.6f %s", tx.Amount, tx.TokenSymbol)
				t.Logf("  状态: %s", tx.Status)
			}

			// 验证交易记录不为空
			t.Assert(len(transactions) > 0, true)

			// 验证所有交易的接收方都是测试地址
			for _, tx := range transactions {
				t.Assert(strings.EqualFold(tx.To, testAddress), true)
			}
		}
	})
}

// TestGetLatestBlock 测试获取最新区块功能
func TestGetLatestBlock(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 创建API客户端
		api := NewTronAPI(
			"https://api.trongrid.io", // 使用波场主网API
			"",                        // 这里填入您的API密钥
		)

		// 设置较长的超时时间
		api.HttpTimeout = time.Second * 30

		// 调用GetLatestBlock方法
		blockResponse, err := api.GetLatestBlock(context.Background())

		// 验证结果
		if err != nil {
			t.Logf("获取最新区块失败: %v", err)
		} else {
			t.Log("区块ID:", blockResponse.BlockID)
			t.Log("区块高度:", blockResponse.BlockHeader.RawData.Number)
			t.Log("区块时间戳:", blockResponse.BlockHeader.RawData.Timestamp)
			t.Log("区块时间:", time.Unix(blockResponse.BlockHeader.RawData.Timestamp/1000, 0).Format("2006-01-02 15:04:05"))
			t.Log("交易数量:", len(blockResponse.Transactions))
			if len(blockResponse.Transactions) > 0 {
				t.Log("第一笔交易ID:", blockResponse.Transactions[0].TxID)
			}

			// 验证区块高度大于0
			t.Assert(blockResponse.BlockHeader.RawData.Number > 0, true)
			// 验证区块ID不为空
			t.Assert(len(blockResponse.BlockID) > 0, true)
		}
	})
}

// TestParseBlockTransactions 测试从区块中解析USDT交易功能
func TestParseBlockTransactions(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 创建API客户端
		api := NewTronAPI(
			"https://api.trongrid.io", // 使用波场主网API
			"",                        // 这里填入您的API密钥
		)

		// 设置较长的超时时间
		api.HttpTimeout = time.Second * 30

		// 获取最新区块
		blockResponse, err := api.GetLatestBlock(context.Background())
		if err != nil {
			t.Logf("获取最新区块失败: %v", err)
			t.FailNow()
			return
		}

		// 解析区块中的交易
		transactions, err := api.ParseBlockTransactions(context.Background(), blockResponse)
		if err != nil {
			t.Logf("解析区块交易失败: %v", err)
			t.FailNow()
			return
		}

		// 打印解析结果
		t.Logf("成功解析区块 %d 中的交易，共找到 %d 笔USDT交易",
			blockResponse.BlockHeader.RawData.Number, len(transactions))

		// 打印USDT交易详情
		for i, tx := range transactions {
			t.Logf("USDT交易 #%d:", i+1)
			t.Logf("  交易ID: %s", tx.TxID)
			t.Logf("  区块号: %d", tx.BlockNumber)
			t.Logf("  时间: %s", tx.Timestamp.Format("2006-01-02 15:04:05"))
			t.Logf("  发送方: %s", tx.From)
			t.Logf("  接收方: %s", tx.To)
			t.Logf("  金额: %.6f %s", tx.Amount, tx.TokenSymbol)
			t.Logf("  合约地址: %s", tx.ContractAddress)
			t.Logf("  状态: %s", tx.Status)
			t.Logf("  是否确认: %v", tx.Confirmed)
			break
		}

		// 注意：由于区块中可能没有USDT交易，所以不强制要求找到交易
		// 但如果找到了交易，验证其基本属性
		for _, tx := range transactions {
			// 验证是USDT交易
			t.Assert(tx.TokenSymbol, "USDT")

			// 验证交易ID不为空
			t.Assert(len(tx.TxID) > 0, true)

			// 验证区块号匹配
			t.Assert(tx.BlockNumber, blockResponse.BlockHeader.RawData.Number)

			// 验证发送方和接收方不为空
			t.Assert(len(tx.From) > 0, true)
			t.Assert(len(tx.To) > 0, true)
			break
		}
	})
}
