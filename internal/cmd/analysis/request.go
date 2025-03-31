package analysis

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/shengdoushi/base58"
	"golang.org/x/time/rate"
)

// TronAPI 波场API接口封装
type TronAPI struct {
	BaseURL     string
	APIKey      string
	HttpTimeout time.Duration
	limiter     *rate.Limiter // 使用官方限流器
	limiterMu   sync.Mutex    // 用于保护limiter的互斥锁
}

// NewTronAPI 创建新的波场API客户端
func NewTronAPI(baseURL, apiKey string) *TronAPI {
	// 创建限流器，默认每秒15个请求，最多允许10个请求排队
	limiter := rate.NewLimiter(rate.Limit(15), 10)

	return &TronAPI{
		BaseURL:     baseURL,
		APIKey:      apiKey,
		HttpTimeout: time.Second * 10,
		limiter:     limiter,
	}
}

// SetRateLimit 设置API请求速率限制
func (t *TronAPI) SetRateLimit(requestsPerSecond int) {
	t.limiterMu.Lock()
	defer t.limiterMu.Unlock()
	t.limiter = rate.NewLimiter(rate.Limit(requestsPerSecond), 1)
}

// maskAPIKey 对API Key进行掩码处理，只显示前4位和后4位
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return apiKey
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}

// doRequest 执行HTTP请求并应用限流
func (t *TronAPI) doRequest(ctx context.Context, req *http.Request) (*http.Response, []byte, error) {
	// 应用限流
	t.limiterMu.Lock()
	limiter := t.limiter
	t.limiterMu.Unlock()

	// 等待获取令牌，如果无法获取则会阻塞
	if err := limiter.Wait(ctx); err != nil {
		return nil, nil, fmt.Errorf("等待限流器超时: %v", err)
	}

	// 发送请求
	client := &http.Client{
		Timeout: t.HttpTimeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("读取响应失败: %v", err)
	}

	// 检查是否是限流响应
	if resp.StatusCode == 403 {
		var errorResp struct {
			Success    bool   `json:"Success"`
			Error      string `json:"Error"`
			StatusCode int    `json:"StatusCode"`
		}

		if err := json.Unmarshal(respBody, &errorResp); err == nil {
			if strings.Contains(errorResp.Error, "exceeds the frequency limit") {
				// 从错误消息中提取暂停时间
				var suspendTime int
				if _, err := fmt.Sscanf(errorResp.Error, "The key exceeds the frequency limit(15), and the query server is suspended for %ds", &suspendTime); err == nil && suspendTime > 0 {
					log.Printf("API Key [%s] 被限流，服务暂停 %d 秒", maskAPIKey(t.APIKey), suspendTime)

					// 动态调整限流器速率 - 暂时降低速率
					t.limiterMu.Lock()
					t.limiter.SetLimit(rate.Limit(0.1)) // 降低到每10秒1个请求
					t.limiterMu.Unlock()

					// 等待指定的暂停时间
					select {
					case <-ctx.Done():
						return nil, nil, ctx.Err()
					case <-time.After(time.Duration(suspendTime) * time.Second):
						// 继续执行
					}

					// 重新创建请求并重试
					newReq := req.Clone(ctx)
					return t.doRequest(ctx, newReq)
				}
			}
		}
	}

	return resp, respBody, nil
}

// GetBlock 获取指定高度的区块
func (t *TronAPI) GetBlock(ctx context.Context, blockNum int64) (Block, error) {
	// TODO: 实现获取区块的API调用
	return Block{}, nil
}

// GetTransaction 获取交易详情
func (t *TronAPI) GetTransaction(ctx context.Context, txID string) (Transaction, error) {
	// TODO: 实现获取交易详情的API调用
	return Transaction{}, nil
}

// GetAddressTransactions 获取地址的交易历史
func (t *TronAPI) GetAddressTransactions(ctx context.Context, address string, limit int) ([]Transaction, error) {
	// TODO: 实现获取地址交易历史的API调用
	return nil, nil
}

// GetTokenBalance 获取地址的代币余额
func (t *TronAPI) GetTokenBalance(ctx context.Context, address, tokenContract string) (float64, error) {
	// 将Base58地址转换为十六进制地址
	hexAddress, err := base58ToHexAddress(address)
	if err != nil {
		return 0, fmt.Errorf("地址转换失败: %v", err)
	}

	// 将合约地址转换为十六进制
	hexContractAddress, err := base58ToHexAddress(tokenContract)
	if err != nil {
		return 0, fmt.Errorf("合约地址转换失败: %v", err)
	}

	// 构建ABI编码的函数调用参数
	parameter := fmt.Sprintf("000000000000000000000000%s", hexAddress[2:]) // 移除41前缀并填充

	// 构建请求体
	requestBody := map[string]interface{}{
		"contract_address":  hexContractAddress,
		"function_selector": "balanceOf(address)",
		"parameter":         parameter,
		"owner_address":     hexAddress,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return 0, fmt.Errorf("构建请求体失败: %v", err)
	}

	// 构建请求
	url := fmt.Sprintf("%s/wallet/triggerconstantcontract", t.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestJSON)))
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if t.APIKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", t.APIKey)
	}

	// 使用限流机制发送请求
	_, respBody, err := t.doRequest(ctx, req)
	if err != nil {
		return 0, err
	}

	// 解析响应
	var response struct {
		ConstantResult []string `json:"constant_result"`
		Result         struct {
			Result bool `json:"result"`
		} `json:"result"`
		Message string `json:"message"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return 0, fmt.Errorf("解析响应失败: %v", err)
	}

	// 检查是否有错误消息
	if response.Message != "" {
		return 0, fmt.Errorf("API返回错误: %s", response.Message)
	}

	// 检查调用是否成功
	if !response.Result.Result {
		return 0, fmt.Errorf("合约调用失败")
	}

	// 解析余额结果
	if len(response.ConstantResult) == 0 {
		return 0, fmt.Errorf("未返回余额结果")
	}

	// 解析十六进制余额
	balanceHex := response.ConstantResult[0]
	if len(balanceHex) < 2 {
		return 0, fmt.Errorf("余额格式错误")
	}

	// 移除可能的0x前缀
	if strings.HasPrefix(balanceHex, "0x") {
		balanceHex = balanceHex[2:]
	}

	// 将十六进制转换为大整数
	balance, ok := new(big.Int).SetString(balanceHex, 16)
	if !ok {
		return 0, fmt.Errorf("解析余额失败: %s", balanceHex)
	}

	// 将余额转换为浮点数，考虑小数位数
	decimals := big.NewInt(1000000) // USDT是6位小数
	balanceFloat := new(big.Float).SetInt(balance)
	divisor := new(big.Float).SetInt(decimals)
	result := new(big.Float).Quo(balanceFloat, divisor)

	finalBalance, _ := result.Float64()
	return finalBalance, nil
}

// base58ToHexAddress 将Base58格式的波场地址转换为十六进制格式
func base58ToHexAddress(base58Address string) (string, error) {
	// 1. Base58解码
	decoded, err := base58.Decode(base58Address, base58.BitcoinAlphabet)
	if err != nil {
		return "", fmt.Errorf("Base58解码失败: %v", err)
	}

	// 2. 检查长度是否合理（地址+校验和）
	if len(decoded) < 4 {
		return "", fmt.Errorf("解码后的地址长度不正确")
	}

	// 3. 分离地址和校验和
	addressBytes := decoded[:len(decoded)-4]
	checksumBytes := decoded[len(decoded)-4:]

	// 4. 验证校验和
	firstSHA := sha256.Sum256(addressBytes)
	secondSHA := sha256.Sum256(firstSHA[:])
	expectedChecksum := secondSHA[:4]

	// 5. 比较校验和
	for i := 0; i < 4; i++ {
		if checksumBytes[i] != expectedChecksum[i] {
			return "", fmt.Errorf("校验和不匹配，地址可能无效")
		}
	}

	// 6. 转换为十六进制格式
	hexAddress := hex.EncodeToString(addressBytes)

	return hexAddress, nil
}

// GetTransactionCount 获取地址的交易数量
func (t *TronAPI) GetTransactionCount(ctx context.Context, address, tokenContract string) (int, error) {
	// TODO: 实现获取交易数量的API调用
	return 0, nil
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

// GetUSDTTransactions 获取地址的USDT交易记录
func (t *TronAPI) GetUSDTTransactions(ctx context.Context, address string, limit int, startTimestamp int64) ([]TRC20Transaction, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s/v1/accounts/%s/transactions/trc20", t.BaseURL, address)

	// 构建查询参数
	query := make(map[string]string)
	query["limit"] = fmt.Sprintf("%d", limit)
	if startTimestamp > 0 {
		query["min_timestamp"] = fmt.Sprintf("%d", startTimestamp)
	}

	// 添加USDT合约地址过滤
	query["contract_address"] = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

	// 构建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加查询参数
	q := req.URL.Query()
	for k, v := range query {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	// 添加API密钥
	if t.APIKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", t.APIKey)
	}
	req.Header.Set("Accept", "application/json")

	// 使用限流机制发送请求
	_, respBody, err := t.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// 解析响应
	var response struct {
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

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 响应: %s", err, string(respBody))
	}

	// 检查API调用是否成功
	if !response.Success {
		return nil, fmt.Errorf("API调用失败")
	}

	// 转换为TRC20Transaction结构
	var transactions []TRC20Transaction
	for _, tx := range response.Data {
		// 解析金额
		amount := "0"
		if tx.Value != "" {
			amount = tx.Value
		}

		// 将金额转换为大整数
		amountBig, ok := new(big.Int).SetString(amount, 10)
		if !ok {
			return nil, fmt.Errorf("解析金额失败: %s", amount)
		}

		// 计算实际金额（考虑小数位数）
		decimals := tx.TokenInfo.Decimals
		if decimals == 0 {
			decimals = 6 // USDT默认为6位小数
		}

		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
		amountFloat := new(big.Float).SetInt(amountBig)
		divisorFloat := new(big.Float).SetInt(divisor)
		result := new(big.Float).Quo(amountFloat, divisorFloat)

		finalAmount, _ := result.Float64()

		// 转换时间戳为时间对象
		timestamp := time.Unix(tx.BlockTimestamp/1000, 0)

		transactions = append(transactions, TRC20Transaction{
			TransactionID:  tx.TransactionID,
			BlockTimestamp: tx.BlockTimestamp,
			From:           tx.From,
			To:             tx.To,
			Amount:         finalAmount,
			TokenName:      tx.TokenInfo.Name,
			TokenSymbol:    tx.TokenInfo.Symbol,
			TokenDecimals:  tx.TokenInfo.Decimals,
			Status:         tx.Status,
			Confirmed:      tx.Approved,
			Timestamp:      timestamp,
		})
	}

	return transactions, nil
}

// GetIncomingUSDTTransactions 获取地址的USDT转入交易记录
func (t *TronAPI) GetIncomingUSDTTransactions(ctx context.Context, address string, limit int, startTimestamp int64) ([]TRC20Transaction, error) {
	// 构建请求URL
	url := fmt.Sprintf("%s/v1/accounts/%s/transactions/trc20", t.BaseURL, address)

	// 构建查询参数
	query := make(map[string]string)
	query["limit"] = fmt.Sprintf("%d", limit)
	if startTimestamp > 0 {
		query["min_timestamp"] = fmt.Sprintf("%d", startTimestamp)
	}

	// 添加USDT合约地址过滤
	query["contract_address"] = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

	// 使用only_to参数只获取转入交易
	query["only_to"] = "true"

	// 构建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加查询参数
	q := req.URL.Query()
	for k, v := range query {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	// 添加API密钥
	if t.APIKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", t.APIKey)
	}
	req.Header.Set("Accept", "application/json")

	// 使用限流机制发送请求
	_, respBody, err := t.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// 解析响应
	var response struct {
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

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 响应: %s", err, string(respBody))
	}

	// 检查API调用是否成功
	if !response.Success {
		return nil, fmt.Errorf("API调用失败")
	}

	// 转换为TRC20Transaction结构
	var transactions []TRC20Transaction
	for _, tx := range response.Data {
		// 解析金额
		amount := "0"
		if tx.Value != "" {
			amount = tx.Value
		}

		// 将金额转换为大整数
		amountBig, ok := new(big.Int).SetString(amount, 10)
		if !ok {
			return nil, fmt.Errorf("解析金额失败: %s", amount)
		}

		// 计算实际金额（考虑小数位数）
		decimals := tx.TokenInfo.Decimals
		if decimals == 0 {
			decimals = 6 // USDT默认为6位小数
		}

		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
		amountFloat := new(big.Float).SetInt(amountBig)
		divisorFloat := new(big.Float).SetInt(divisor)
		result := new(big.Float).Quo(amountFloat, divisorFloat)

		finalAmount, _ := result.Float64()

		// 转换时间戳为时间对象
		timestamp := time.Unix(tx.BlockTimestamp/1000, 0)

		transactions = append(transactions, TRC20Transaction{
			TransactionID:  tx.TransactionID,
			BlockTimestamp: tx.BlockTimestamp,
			From:           tx.From,
			To:             tx.To,
			Amount:         finalAmount,
			TokenName:      tx.TokenInfo.Name,
			TokenSymbol:    tx.TokenInfo.Symbol,
			TokenDecimals:  tx.TokenInfo.Decimals,
			Status:         tx.Status,
			Confirmed:      tx.Approved,
			Timestamp:      timestamp,
		})
	}

	return transactions, nil
}

// GetIncomingUSDTTransactionsByTimeRange 根据时间范围获取USDT转入交易记录
func (t *TronAPI) GetIncomingUSDTTransactionsByTimeRange(ctx context.Context, address string, startTime, endTime time.Time, limit int) ([]TRC20Transaction, error) {
	// 将时间转换为毫秒时间戳
	startTimestamp := startTime.UnixNano() / int64(time.Millisecond)

	// 获取转入交易记录
	transactions, err := t.GetIncomingUSDTTransactions(ctx, address, limit, startTimestamp)
	if err != nil {
		return nil, err
	}

	// 过滤结束时间之后的交易
	endTimestamp := endTime.UnixNano() / int64(time.Millisecond)
	var filteredTransactions []TRC20Transaction

	for _, tx := range transactions {
		if tx.BlockTimestamp <= endTimestamp {
			filteredTransactions = append(filteredTransactions, tx)
		}
	}

	return filteredTransactions, nil
}

// GetUSDTTransactionsByTimeRange 根据时间范围获取USDT交易记录
func (t *TronAPI) GetUSDTTransactionsByTimeRange(ctx context.Context, address string, startTime, endTime time.Time, limit int) ([]TRC20Transaction, error) {
	// 将时间转换为毫秒时间戳
	startTimestamp := startTime.UnixNano() / int64(time.Millisecond)

	// 获取交易记录
	transactions, err := t.GetUSDTTransactions(ctx, address, limit, startTimestamp)
	if err != nil {
		return nil, err
	}

	// 过滤结束时间之后的交易
	endTimestamp := endTime.UnixNano() / int64(time.Millisecond)
	var filteredTransactions []TRC20Transaction

	for _, tx := range transactions {
		if tx.BlockTimestamp <= endTimestamp {
			filteredTransactions = append(filteredTransactions, tx)
		}
	}

	return filteredTransactions, nil
}
