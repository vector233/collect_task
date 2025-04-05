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
		transactions, err := api.GetUSDTTransactions(context.Background(), testAddress, 10, 0, false)

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
		block, err := api.GetLatestBlock(context.Background())

		// 验证结果
		if err != nil {
			t.Logf("获取最新区块失败: %v", err)
		} else {
			t.Log("区块ID:", block.BlockID)
			t.Log("区块高度:", block.BlockNumber)
			t.Log("区块时间:", block.BlockTime.Format("2006-01-02 15:04:05"))
			t.Log("交易数量:", block.TransactionNum)
			if block.TransactionNum > 0 && len(block.Transactions) > 0 {
				t.Log("第一笔交易ID:", block.Transactions[0])
			}

			// 验证区块高度大于0
			t.Assert(block.BlockNumber > 0, true)
			// 验证区块ID不为空
			t.Assert(len(block.BlockID) > 0, true)
		}
	})
}

// TestGetTransaction 测试获取交易详情功能
func TestGetTransaction(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 创建API客户端
		api := NewTronAPI(
			"https://api.trongrid.io", // 使用波场主网API
			"",                        // 这里填入您的API密钥
		)

		// 设置较长的超时时间
		api.HttpTimeout = time.Second * 30

		// 使用一个已知的USDT交易ID进行测试
		// 这是一个真实的USDT转账交易
		txID := "33bd8c7d1ae3fe4e7525fa05e4e0d37fd4463a8fddac797c7531d71c2dbf038e"

		// 调用GetTransaction方法
		tx, err := api.GetTransaction(context.Background(), txID)

		// 验证结果
		if err != nil {
			// 如果是"未找到交易"错误，可能是测试交易ID已经过期或不存在
			if strings.Contains(err.Error(), "未找到交易") {
				t.Log("测试交易ID不存在，请更新为有效的交易ID")
			} else {
				t.Errorf("获取交易详情失败: %v", err)
			}
		} else {
			t.Log("交易ID:", tx.TxID)
			t.Log("区块号:", tx.BlockNumber)
			t.Log("时间:", tx.Timestamp.Format("2006-01-02 15:04:05"))
			t.Log("发送方:", tx.From)
			t.Log("接收方:", tx.To)
			t.Log("金额:", tx.Amount)
			t.Log("代币:", tx.TokenSymbol)
			t.Log("合约地址:", tx.ContractAddress)
			t.Log("合约类型:", tx.ContractType)
			t.Log("状态:", tx.Status)
			t.Log("是否确认:", tx.Confirmed)

			// 验证交易ID匹配
			t.Assert(tx.TxID, txID)
		}
	})
}

// TestGetUSDTTransaction 测试获取USDT交易详情
func TestGetUSDTTransaction(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 创建API客户端
		api := NewTronAPI(
			"https://api.trongrid.io", // 使用波场主网API
			"",                        // 这里填入您的API密钥
		)

		// 设置较长的超时时间
		api.HttpTimeout = time.Second * 30

		// 先获取一个地址的USDT交易记录
		testAddress := "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf"
		transactions, err := api.GetUSDTTransactions(context.Background(), testAddress, 1, 0, false)

		if err != nil || len(transactions) == 0 {
			t.Log("无法获取测试交易，跳过测试")
			return
		}

		// 使用获取到的第一笔交易ID
		txID := transactions[0].TransactionID
		t.Log("测试交易ID:", txID)

		// 调用GetTransaction方法
		tx, err := api.GetTransaction(context.Background(), txID)

		// 验证结果
		if err != nil {
			t.Errorf("获取USDT交易详情失败: %v", err)
		} else {
			t.Log("交易ID:", tx.TxID)
			t.Log("区块号:", tx.BlockNumber)
			t.Log("时间:", tx.Timestamp.Format("2006-01-02 15:04:05"))
			t.Log("发送方:", tx.From)
			t.Log("接收方:", tx.To)
			t.Log("金额:", tx.Amount)
			t.Log("代币:", tx.TokenSymbol)
			t.Log("合约地址:", tx.ContractAddress)
			t.Log("合约类型:", tx.ContractType)
			t.Log("状态:", tx.Status)
			t.Log("是否确认:", tx.Confirmed)

			// 验证交易ID匹配
			t.Assert(tx.TxID, txID)

			// 验证是USDT交易
			t.Assert(tx.TokenSymbol, "USDT")
			t.Assert(tx.ContractAddress == "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" ||
				tx.ContractAddress == "41a614f803b6fd780986a42c78ec9c7f77e6ded13c", true)
		}
	})
}

// TestParseContractInfo 测试合约信息解析功能
func TestParseContractInfo(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 创建API客户端
		api := NewTronAPI(
			"https://api.trongrid.io", // 使用波场主网API
			"",                        // 这里填入您的API密钥
		)

		// 设置较长的超时时间
		api.HttpTimeout = time.Second * 30

		// 先获取一个地址的USDT交易记录
		testAddress := "TDqSquXBgUCLYvYC4XZgrprLK589dkhSCf"
		transactions, err := api.GetUSDTTransactions(context.Background(), testAddress, 1, 0, false)

		if err != nil || len(transactions) == 0 {
			t.Log("无法获取测试交易，跳过测试")
			return
		}

		// 使用获取到的第一笔交易ID
		txID := transactions[0].TransactionID

		// 获取交易基本信息
		txResponse, err := api.fetchTransactionBasicInfo(context.Background(), txID)
		if err != nil {
			t.Errorf("获取交易基本信息失败: %v", err)
			return
		}

		// 初始化交易结构
		transaction := Transaction{
			TxID: txID,
		}

		// 测试解析合约信息
		api.parseContractInfo(&transaction, txResponse)

		// 验证结果
		t.Log("合约类型:", transaction.ContractType)
		t.Log("合约地址:", transaction.ContractAddress)
		t.Log("代币名称:", transaction.TokenName)
		t.Log("代币符号:", transaction.TokenSymbol)
		t.Log("发送方:", transaction.From)
		t.Log("接收方:", transaction.To)
		t.Log("金额:", transaction.Amount)

		// 验证是否成功解析了合约信息
		t.Assert(transaction.ContractType != "", true)
		t.Assert(transaction.From != "", true)
	})
}

// TestParseTokenTransferData 测试代币转账数据解析功能
func TestParseTokenTransferData(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 创建API客户端
		api := NewTronAPI(
			"https://api.trongrid.io", // 使用波场主网API
			"",                        // 这里填入您的API密钥
		)

		// 模拟一个代币转账数据
		// 这是一个模拟的transfer方法调用数据
		// a9059cbb: transfer方法的选择器
		// 000000000000000000000000410000000000000000000000000000000000000000: 接收地址(填充到32字节)
		// 0000000000000000000000000000000000000000000000000000000000989680: 金额(10 USDT = 10000000)
		// data := "a9059cbb0000000000000000000000004100000000000000000000000000000000000000000000000000000000000000000000000000000000000000989680"
		data := "a9059cbb000000000000000000000041256e0a6472521d81239e8ee6fd29bb5cd189b5e9000000000000000000000000000000000000000000000000000000001daee080"

		// 初始化交易结构
		transaction := Transaction{
			TxID: "test_tx_id",
		}

		// 测试解析代币转账数据
		api.parseTokenTransferData(&transaction, data)

		// 验证结果
		t.Log("接收方:", transaction.To)
		t.Log("金额:", transaction.Amount)

		// 验证金额是否正确解析
		// 10000000 / 1000000 = 10.0 USDT
		t.Assert(transaction.Amount, 498.0)
	})
}

// TestGetTransactionWithLocalNode 测试使用本地节点获取交易详情
func TestGetTransactionWithLocalNode(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// 跳过此测试，除非明确要测试自建节点
		t.Skip("跳过自建节点测试")

		// 创建API客户端，使用本地节点
		api := NewTronAPI(
			"http://104.233.192.15:8090", // 自建节点地址
			"",
		)

		// 设置较长的超时时间
		api.HttpTimeout = time.Second * 30

		// 使用一个已知的交易ID进行测试
		txID := "5f92e8f3245b2e5b8a3bf1b9a0a5d1d1f94a0c6c9a5f92e8f3245b2e5b8a3bf1"

		// 调用GetTransaction方法
		tx, err := api.GetTransaction(context.Background(), txID)

		// 验证结果
		if err != nil {
			// 如果是"未找到交易"错误，可能是测试交易ID已经过期或不存在
			if strings.Contains(err.Error(), "未找到交易") {
				t.Log("测试交易ID不存在，请更新为有效的交易ID")
			} else {
				t.Errorf("获取交易详情失败: %v", err)
			}
		} else {
			t.Log("交易ID:", tx.TxID)
			t.Log("区块号:", tx.BlockNumber)
			t.Log("时间:", tx.Timestamp.Format("2006-01-02 15:04:05"))
			t.Log("发送方:", tx.From)
			t.Log("接收方:", tx.To)
			t.Log("金额:", tx.Amount)
			t.Log("代币:", tx.TokenSymbol)
			t.Log("合约地址:", tx.ContractAddress)
			t.Log("合约类型:", tx.ContractType)
			t.Log("状态:", tx.Status)
			t.Log("是否确认:", tx.Confirmed)

			// 验证交易ID匹配
			t.Assert(tx.TxID, txID)
		}
	})
}
