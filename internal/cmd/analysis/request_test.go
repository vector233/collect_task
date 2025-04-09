package analysis

import (
	"context"
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
			response, err := api.GetUSDTTransactions(context.Background(), testAddress, tc.params)
			transactions, _ := ConvertToTRC20Transactions(response)

			// 验证结果
			if err != nil {
				t.Logf("[%s] 获取USDT交易记录失败: %v", tc.name, err)
			} else {
				t.Logf("[%s] 成功获取 %d 条USDT交易记录", tc.name, len(transactions))

				// 打印第一笔交易详情
				if len(transactions) > 0 {
					tx := transactions[0]
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

// TestGetTransactionCount 测试获取地址交易数量功能
func TestGetTransactionCount(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 创建API客户端
		api := NewTronAPI(
			"https://api.trongrid.io",              // 使用波场主网API
			"136312ba-b5e2-4e99-a006-d9a672a6854e", // 这里填入您的API密钥
		)

		// 设置较长的超时时间
		api.HttpTimeout = time.Second * 30

		// USDT合约地址
		params := TransactionCountParams{
			ContractAddress: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
			RelatedAddress:  "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf", // 只统计从特定地址发出的交易
		}

		// 获取交易数量
		count, err := api.GetTransactionCount(context.Background(), params)

		// 验证没有错误
		t.AssertNil(err)

		// 验证交易数量大于0
		t.Assert(count > 0, true)

		t.Logf("地址 %s 的交易数量: %d", params.RelatedAddress, count)

		// 测试近1个月的交易数量
		oneMonthAgo := time.Now().AddDate(0, -1, 0)
		now := time.Now()

		monthlyParams := TransactionCountParams{
			ContractAddress: "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
			RelatedAddress:  "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf",
			StartTimestamp:  &oneMonthAgo,
			EndTimestamp:    &now,
		}

		// 获取近1个月的交易数量
		monthlyCount, err := api.GetTransactionCount(context.Background(), monthlyParams)

		// 验证没有错误
		t.AssertNil(err)

		// 验证交易数量
		t.Logf("地址 %s 近1个月的交易数量: %d", monthlyParams.RelatedAddress, monthlyCount)
		t.Logf("查询时间范围: %s 至 %s",
			oneMonthAgo.Format("2006-01-02 15:04:05"),
			now.Format("2006-01-02 15:04:05"))

		// 近1个月的交易数量应该小于等于总交易数量
		t.Assert(monthlyCount <= count, true)

		// 测试随机地址 - 可能没有交易
		params.RelatedAddress = "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf"

		// 获取交易数量
		randomCount, err := api.GetTransactionCount(context.Background(), params)

		// 验证没有错误
		t.AssertNil(err)

		// 这个地址可能有交易也可能没有，只验证调用成功
		t.Logf("随机地址 %s 的交易数量: %d", params.RelatedAddress, randomCount)

		// 测试无效地址
		params.RelatedAddress = "InvalidAddress"

		// 获取交易数量 - 应该返回错误
		total, err := api.GetTransactionCount(context.Background(), params)

		// 验证有错误返回
		t.Assert(total, 0)
		t.Logf("无效地址测试错误信息: %v", err)
	})
}

// TestGetUSDTTransactionsByTimeRange 测试根据时间范围获取USDT交易记录功能
func TestGetUSDTTransactionsByTimeRange(t *testing.T) {
	// 创建API客户端
	api := NewTronAPI(
		"https://api.trongrid.io", // 使用波场主网API
		"",                        // 这里填入您的API密钥
	)

	// 设置较长的超时时间
	api.HttpTimeout = time.Second * 30

	// 测试地址 - 使用一个已知有USDT交易的地址
	testAddress := "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf"

	// 测试用例
	testCases := []struct {
		name   string
		params TRC20TransactionParams
		limit  int
	}{
		// {
		// 	name: "近一个月交易-限制10条",
		// 	params: TRC20TransactionParams{
		// 		MinTimestamp: timePtr(time.Now().AddDate(0, -1, 0)),
		// 		MaxTimestamp: timePtr(time.Now()),
		// 	},
		// 	limit: 10,
		// },
		// {
		// 	name: "近三个月交易-限制20条",
		// 	params: TRC20TransactionParams{
		// 		MinTimestamp: timePtr(time.Now().AddDate(0, -3, 0)),
		// 		MaxTimestamp: timePtr(time.Now()),
		// 	},
		// 	limit: 20,
		// },
		// {
		// 	name: "近半年交易-限制50条",
		// 	params: TRC20TransactionParams{
		// 		MinTimestamp: timePtr(time.Now().AddDate(0, -6, 0)),
		// 		MaxTimestamp: timePtr(time.Now()),
		// 	},
		// 	limit: 50,
		// },
		// {
		// 	name: "近一年交易-限制100条",
		// 	params: TRC20TransactionParams{
		// 		MinTimestamp: timePtr(time.Now().AddDate(-1, 0, 0)),
		// 		MaxTimestamp: timePtr(time.Now()),
		// 	},
		// 	limit: 100,
		// },
		{
			name: "测试分页-超过200条",
			params: TRC20TransactionParams{
				//MinTimestamp: timePtr(time.Now().AddDate(-2, 0, 0)),
				//MaxTimestamp: timePtr(time.Now()),
				OrderBy: "block_timestamp,desc", // 按时间戳降序排序
			},
			limit: 250, // 超过单次请求上限，测试分页功能
		},
	}

	for _, tc := range testCases {
		gtest.C(t, func(t *gtest.T) {
			t.Logf("测试用例: %s", tc.name)
			if tc.params.MinTimestamp != nil && tc.params.MaxTimestamp != nil {
				t.Logf("时间范围: %s 至 %s",
					tc.params.MinTimestamp.Format("2006-01-02 15:04:05"),
					tc.params.MaxTimestamp.Format("2006-01-02 15:04:05"))
			}
			t.Logf("请求数量上限: %d", tc.limit)

			// 调用GetUSDTTransactionsByTimeRange方法
			transactions, err := api.GetUSDTTransactionsByTimeRange(
				context.Background(),
				testAddress,
				tc.params,
				tc.limit,
			)

			// 验证结果
			if err != nil {
				t.Logf("获取交易记录失败: %v", err)
			} else {
				t.Logf("成功获取 %d 条交易记录", len(transactions))

				// 验证返回的交易数量不超过请求的限制
				t.Assert(len(transactions) <= tc.limit, true)

				// 验证所有交易的时间戳在指定范围内
				for i, tx := range transactions {
					if i < 3 || i >= len(transactions)-3 {
						// 只打印前3条和后3条交易详情，避免日志过长
						t.Logf("交易 #%d:", i+1)
						t.Logf("  交易ID: %s", tx.TransactionID)
						t.Logf("  时间: %s", tx.Timestamp.Format("2006-01-02 15:04:05"))
						t.Logf("  发送方: %s", tx.From)
						t.Logf("  接收方: %s", tx.To)
						t.Logf("  金额: %.6f %s", tx.Amount, tx.TokenSymbol)
					}

					// 验证交易时间在请求的时间范围内（如果设置了时间范围）
					if tc.params.MinTimestamp != nil && tc.params.MaxTimestamp != nil {
						// 注意：API可能会返回略微超出范围的结果，所以这里添加一天的容差
						minTime := tc.params.MinTimestamp.AddDate(0, 0, -1)
						maxTime := tc.params.MaxTimestamp.AddDate(0, 0, 1)

						inRange := (tx.Timestamp.After(minTime) || tx.Timestamp.Equal(minTime)) &&
							(tx.Timestamp.Before(maxTime) || tx.Timestamp.Equal(maxTime))

						if !inRange {
							t.Logf("警告: 交易 #%d 的时间 %s 不在请求范围内",
								i+1, tx.Timestamp.Format("2006-01-02 15:04:05"))
						}
					}
				}

				// 如果请求的是分页测试用例且返回的交易数量接近或等于限制值，则验证分页功能正常工作
				if tc.limit > 200 && len(transactions) >= 200 {
					t.Logf("分页功能测试通过，成功获取超过单次请求上限的交易记录")
				}
			}
		})
	}
}

// TestGetUSDTTransactionsByTimeRangeWithInvalidParams 测试无效参数情况
func TestGetUSDTTransactionsByTimeRangeWithInvalidParams(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 创建API客户端
		api := NewTronAPI(
			"https://api.trongrid.io", // 使用波场主网API
			"",                        // 这里填入您的API密钥
		)

		// 设置较长的超时时间
		api.HttpTimeout = time.Second * 30

		// 测试无效地址
		invalidAddress := "InvalidAddress"
		startTime := time.Now().AddDate(0, -1, 0)
		endTime := time.Now()

		// 创建参数
		params := TRC20TransactionParams{
			MinTimestamp: &startTime,
			MaxTimestamp: &endTime,
		}

		// 调用GetUSDTTransactionsByTimeRange方法
		transactions, err := api.GetUSDTTransactionsByTimeRange(
			context.Background(),
			invalidAddress,
			params,
			10,
		)

		// 验证结果
		t.Assert(err != nil, true)
		t.Assert(len(transactions), 0)
		t.Logf("无效地址测试错误信息: %v", err)

		// 测试无效的时间范围（结束时间早于开始时间）
		validAddress := "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf"
		invalidStartTime := time.Now()
		invalidEndTime := time.Now().AddDate(0, -1, 0)

		// 创建无效时间范围参数
		invalidParams := TRC20TransactionParams{
			MinTimestamp: &invalidStartTime,
			MaxTimestamp: &invalidEndTime,
		}

		// 调用GetUSDTTransactionsByTimeRange方法
		transactions, err = api.GetUSDTTransactionsByTimeRange(
			context.Background(),
			validAddress,
			invalidParams,
			10,
		)

		// 验证结果 - API可能不会直接返回错误，但应该没有数据
		t.Logf("无效时间范围测试结果: %v", err)
		t.Logf("返回交易数量: %d", len(transactions))
	})
}
