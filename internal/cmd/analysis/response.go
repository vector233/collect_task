package analysis

import (
	"context"
	"math/big"
	"strings"
	"time"

	"tron-lion/utility"
)

// ParseBlockTransactions 将区块中的交易解析为Transaction结构体
func ParseBlockTransactions(ctx context.Context, blockResponse BlockResponse) ([]Transaction, error) {
	var transactions []Transaction

	for _, tx := range blockResponse.Transactions {
		// 检查是否有合约调用
		if len(tx.RawData.Contract) == 0 {
			continue
		}

		// 初始化交易结构
		transaction := Transaction{
			TxID:           tx.TxID,
			BlockNumber:    blockResponse.BlockHeader.RawData.Number,
			BlockTimestamp: blockResponse.BlockHeader.RawData.Timestamp,
			Timestamp:      time.Unix(blockResponse.BlockHeader.RawData.Timestamp/1000, 0),
		}

		// 设置交易状态
		if len(tx.Ret) > 0 {
			transaction.Status = tx.Ret[0].ContractRet
			transaction.Confirmed = tx.Ret[0].ContractRet == "SUCCESS"
		}

		contract := tx.RawData.Contract[0]
		transaction.ContractType = contract.Type

		// 检查是否是USDT交易
		isUSDT := false

		// 根据合约类型处理不同的交易
		if contract.Type == ContractTypeTriggerSmart {
			// 获取合约地址
			contractAddress := contract.Parameter.Value.ContractAddress
			transaction.ContractAddress = contractAddress

			// 检查是否是USDT合约地址
			if contractAddress == "41a614f803b6fd780986a42c78ec9c7f77e6ded13c" || // 十六进制
				contractAddress == "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" { // Base58
				isUSDT = true
				transaction.TokenName = "Tether USD"
				transaction.TokenSymbol = "USDT"
			}

			// 获取调用者地址
			ownerAddress := contract.Parameter.Value.OwnerAddress
			transaction.From = ownerAddress

			// 获取调用数据
			data := contract.Parameter.Value.Data

			// 检查是否是transfer方法 (0xa9059cbb)
			if len(data) >= 8 && strings.HasPrefix(data, "a9059cbb") {
				// 确认是USDT转账
				if isUSDT {
					// 提取接收地址
					if len(data) >= 72 {
						toAddrHex := "41" + data[32:72]
						toAddr, err := utility.HexAddressToBase58(toAddrHex)
						if err == nil {
							transaction.To = toAddr
						}
					}

					// 提取金额
					if len(data) >= 136 {
						amountHex := data[72:136]
						amountBig, ok := new(big.Int).SetString(amountHex, 16)
						if ok {
							// USDT有6位小数
							divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(6), nil)
							amountFloat := new(big.Float).SetInt(amountBig)
							divisorFloat := new(big.Float).SetInt(divisor)
							result := new(big.Float).Quo(amountFloat, divisorFloat)

							transaction.Amount, _ = result.Float64()
						}
					}
				}
			}
		}

		// 只添加USDT交易
		if isUSDT {
			transactions = append(transactions, transaction)
		}
	}

	return transactions, nil
}
