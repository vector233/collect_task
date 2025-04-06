package analysis

import (
	"context"
	"testing"
	"time"

	"github.com/gogf/gf/v2/test/gtest"
)

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
		transactions, err := ParseBlockTransactions(context.Background(), blockResponse)
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
