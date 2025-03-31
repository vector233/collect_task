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
	gtest.C(t, func(t *gtest.T) {
		// 创建API客户端
		api := NewTronAPI(
			"https://api.trongrid.io", // 使用波场主网API
			"",                        // 这里填入您的API密钥
		)

		// 设置较长的超时时间
		api.HttpTimeout = time.Second * 30

		// 测试地址 - 使用一个已知有USDT交易的地址
		testAddress := "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf"

		// 调用GetUSDTTransactions方法，获取最近10条交易
		transactions, err := api.GetUSDTTransactions(context.Background(), testAddress, 10, 0)

		// 验证结果
		if err != nil {
			t.Logf("获取USDT交易记录失败: %v", err)
		} else {
			t.Logf("成功获取 %d 条USDT交易记录", len(transactions))

			// 打印交易详情
			for i, tx := range transactions {
				t.Logf("交易 #%d:", i+1)
				t.Logf("  交易ID: %s", tx.TransactionID)
				t.Logf("  时间: %s", tx.Timestamp.Format("2006-01-02 15:04:05"))
				t.Logf("  发送方: %s", tx.From)
				t.Logf("  接收方: %s", tx.To)
				t.Logf("  金额: %.6f %s", tx.Amount, tx.TokenSymbol)
				t.Logf("  状态: %s", tx.Status)
			}

			// 验证交易记录不为空
			t.Assert(len(transactions) > 0, true)
		}
	})
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
