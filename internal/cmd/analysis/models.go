package analysis

import (
	"encoding/json"
	"time"
)

// Block 区块信息
type Block struct {
	BlockID        string    `json:"block_id"`
	BlockNumber    int64     `json:"block_number"`
	Timestamp      int64     `json:"timestamp"`
	TransactionNum int       `json:"transaction_num"`
	Transactions   []string  `json:"transactions"`
	Confirmed      bool      `json:"confirmed"`
	BlockTime      time.Time `json:"-"` // 转换后时间
}

// Transaction 交易信息
type Transaction struct {
	TxID            string    `json:"tx_id"`
	BlockNumber     int64     `json:"block_number"`
	BlockTimestamp  int64     `json:"block_timestamp"`
	From            string    `json:"from"`
	To              string    `json:"to"`
	Amount          float64   `json:"amount"`
	TokenName       string    `json:"token_name"`       // 代币名称，如 "Tether USD"
	TokenSymbol     string    `json:"token_symbol"`     // 代币符号，如 "USDT"
	ContractAddress string    `json:"contract_address"` // 合约地址
	ContractType    string    `json:"contract_type"`    // 合约类型，如 "TriggerSmartContract"
	Status          string    `json:"status"`           // 交易状态
	Confirmed       bool      `json:"confirmed"`        // 是否已确认
	Fee             float64   `json:"fee"`              // 交易费用
	Timestamp       time.Time `json:"-"`                // 转换后时间
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

// 活跃地址结构
type ActiveAddress struct {
	Address          string    // 地址
	Balance          float64   // 余额
	TxCount          int       // 交易总数
	LastActiveTime   time.Time // 最后活跃时间
	FrequentOutAddrs []string  // 常转出地址列表
	IsActive         bool      // 是否活跃
	RecursionDepth   int       // 递归深度
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
	Timestamp      time.Time `json:"-"` // 转换后时间
}

// APIErrorResponse 表示API错误响应
type APIErrorResponse struct {
	Success    bool   `json:"Success"`
	Error      string `json:"Error"`
	StatusCode int    `json:"StatusCode"`
}

// TransactionResponse 表示交易详情API响应
type TransactionResponse struct {
	Visible bool   `json:"visible"`
	TxID    string `json:"txID"`
	RawData struct {
		Contract []struct {
			Parameter struct {
				Value   json.RawMessage `json:"value"`
				TypeURL string          `json:"type_url"`
			} `json:"parameter"`
			Type string `json:"type"`
		} `json:"contract"`
		RefBlockBytes string `json:"ref_block_bytes"`
		RefBlockHash  string `json:"ref_block_hash"`
		Expiration    int64  `json:"expiration"`
		Timestamp     int64  `json:"timestamp"`
	} `json:"raw_data"`
	RawDataHex string `json:"raw_data_hex"`
}

// TransactionInfoResponse 表示交易信息API响应
type TransactionInfoResponse struct {
	ID              string   `json:"id"`
	BlockNumber     int64    `json:"blockNumber"`
	BlockTimeStamp  int64    `json:"blockTimeStamp"`
	ContractResult  []string `json:"contractResult"`
	ContractAddress string   `json:"contract_address,omitempty"`
	Receipt         struct {
		EnergyUsage       int64  `json:"energy_usage"`
		EnergyFee         int64  `json:"energy_fee"`
		OriginEnergyUsage int64  `json:"origin_energy_usage"`
		EnergyUsageTotal  int64  `json:"energy_usage_total"`
		NetUsage          int64  `json:"net_usage"`
		NetFee            int64  `json:"net_fee"`
		Result            string `json:"result"`
	} `json:"receipt"`
	Log []struct {
		Address string   `json:"address"`
		Topics  []string `json:"topics"`
		Data    string   `json:"data"`
	} `json:"log,omitempty"`
	Fee int64 `json:"fee"`
}

// TriggerSmartContractParam 表示智能合约调用参数
type TriggerSmartContractParam struct {
	OwnerAddress    string `json:"owner_address"`
	ContractAddress string `json:"contract_address"`
	Data            string `json:"data"`
}

// TransferContractParam 表示TRX转账参数
type TransferContractParam struct {
	OwnerAddress string `json:"owner_address"`
	ToAddress    string `json:"to_address"`
	Amount       int64  `json:"amount"`
}
