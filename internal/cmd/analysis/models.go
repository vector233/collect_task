package analysis

import (
	"encoding/json"
	"time"
)

// Block 表示波场区块信息
type Block struct {
	BlockID        string    `json:"block_id"`
	BlockNumber    int64     `json:"block_number"`
	Timestamp      int64     `json:"timestamp"`
	TransactionNum int       `json:"transaction_num"`
	Transactions   []string  `json:"transactions"`
	Confirmed      bool      `json:"confirmed"`
	BlockTime      time.Time `json:"-"` // 转换后的时间
}

// Transaction 表示交易详情
type Transaction struct {
	TxID           string    `json:"tx_id"`
	BlockNumber    int64     `json:"block_number"`
	BlockTimestamp int64     `json:"block_timestamp"`
	From           string    `json:"from"`
	To             string    `json:"to"`
	Amount         float64   `json:"amount"`
	ContractType   string    `json:"contract_type"`
	Status         string    `json:"status"`
	Timestamp      time.Time `json:"-"` // 转换后的时间
}

// BlockResponse 表示波场区块API响应
type BlockResponse struct {
	BlockID     string `json:"blockID"`
	BlockHeader struct {
		RawData struct {
			Number         int64  `json:"number"`
			TxTrieRoot     string `json:"txTrieRoot"`
			WitnessAddress string `json:"witness_address"`
			ParentHash     string `json:"parentHash"`
			Version        int    `json:"version"`
			Timestamp      int64  `json:"timestamp"`
		} `json:"raw_data"`
		WitnessSignature string `json:"witness_signature"`
	} `json:"block_header"`
	Transactions []struct {
		Ret []struct {
			ContractRet string `json:"contractRet"`
		} `json:"ret"`
		Signature []string `json:"signature"`
		TxID      string   `json:"txID"`
		RawData   struct {
			Contract []struct {
				Parameter struct {
					Value   json.RawMessage `json:"value"`
					TypeURL string          `json:"type_url"`
				} `json:"parameter"`
				Type         string `json:"type"`
				PermissionID int    `json:"Permission_id,omitempty"`
			} `json:"contract"`
			RefBlockBytes string `json:"ref_block_bytes"`
			RefBlockHash  string `json:"ref_block_hash"`
			Expiration    int64  `json:"expiration"`
			Timestamp     int64  `json:"timestamp"`
		} `json:"raw_data"`
		RawDataHex string `json:"raw_data_hex"`
	} `json:"transactions"`
}

// TokenInfo 代币信息
type TokenInfo struct {
	Symbol     string
	Name       string
	Decimals   int
	ContractID string
}

// ActiveAddress 活跃地址信息
type ActiveAddress struct {
	Address    string
	Balance    float64
	TxCount    int
	LastTxTime time.Time
}

// FrequentTransferAddress 常转出地址信息
type FrequentTransferAddress struct {
	Address      string
	Pattern      string // 前X后Y格式
	OutTxCount   int
	LargeTxCount int
	AvgOutAmount float64
	LastTxTime   time.Time
}

// OrderInfo 订单信息
type OrderInfo struct {
	OrderNo                 string
	ActiveAddress           string
	FrequentTransferPattern string
	LastTxTime              time.Time
	Amount                  float64
}

// TokenBalanceResponse 表示代币余额API响应
type TokenBalanceResponse struct {
	ConstantResult []string `json:"constant_result"`
	Result         struct {
		Result bool `json:"result"`
	} `json:"result"`
	Message string `json:"message"`
}

// TRC20TransactionResponse 表示TRC20交易记录API响应
type TRC20TransactionResponse struct {
	Data []struct {
		TransactionID  string `json:"transaction_id"`
		BlockTimestamp int64  `json:"block_timestamp"`
		From           string `json:"from"`
		To             string `json:"to"`
		Value          string `json:"value"`
		TokenInfo      struct {
			Symbol   string `json:"symbol"`
			Name     string `json:"name"`
			Decimals int    `json:"decimals"`
			Address  string `json:"address"`
		} `json:"token_info"`
		Type     string `json:"type"`
		Status   string `json:"status,omitempty"`
		Approved bool   `json:"approved,omitempty"`
	} `json:"data"`
	Success bool `json:"success"`
	Meta    struct {
		At       int64 `json:"at"`
		PageSize int   `json:"page_size"`
	} `json:"meta"`
}

// TRC20Transaction 表示TRC20代币交易记录
type TRC20Transaction struct {
	TransactionID  string    `json:"transaction_id"`
	BlockTimestamp int64     `json:"block_timestamp"`
	From           string    `json:"from"`
	To             string    `json:"to"`
	Amount         float64   `json:"amount"`
	TokenName      string    `json:"token_name"`
	TokenSymbol    string    `json:"token_symbol"`
	TokenDecimals  int       `json:"token_decimals"`
	Status         string    `json:"status"`
	Confirmed      bool      `json:"confirmed"`
	Timestamp      time.Time `json:"-"` // 转换后的时间
}

// APIErrorResponse 表示API错误响应
type APIErrorResponse struct {
	Success    bool   `json:"Success"`
	Error      string `json:"Error"`
	StatusCode int    `json:"StatusCode"`
}
