package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"golang.org/x/time/rate"

	"tron-lion/utility"
)

// TronAPI 波场API封装
type TronAPI struct {
	BaseURL     string
	APIKey      string
	HttpTimeout time.Duration
	limiter     *rate.Limiter // 限流器
	limiterMu   sync.Mutex    // 互斥锁
}

// 创建波场API客户端
func NewTronAPI(baseURL, apiKey string) *TronAPI {
	// 默认限流：15/秒，队列10个
	limiter := rate.NewLimiter(rate.Limit(15), 10)

	return &TronAPI{
		BaseURL:     baseURL,
		APIKey:      apiKey,
		HttpTimeout: time.Second * 10,
		limiter:     limiter,
	}
}

// 设置API限流
func (t *TronAPI) SetRateLimit(requestsPerSecond, bucket int) {
	t.limiterMu.Lock()
	defer t.limiterMu.Unlock()
	t.limiter = rate.NewLimiter(rate.Limit(requestsPerSecond), bucket)
}

// 对Key进行掩码，只保留前4位和后4位
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return apiKey
	}
	return apiKey[:4] + "..." + apiKey[len(apiKey)-4:]
}

// 执行HTTP请求并限流
func (t *TronAPI) doRequest(ctx context.Context, req *http.Request) (*http.Response, []byte, error) {
	// 获取限流器
	t.limiterMu.Lock()
	limiter := t.limiter
	t.limiterMu.Unlock()

	// 打印限流器状态
	tokens := limiter.Tokens()
	limit := limiter.Limit()
	burst := limiter.Burst()
	g.Log().Debugf(ctx, "限流器状态: 当前令牌数=%.2f, 限制=%.2f/秒, 桶容量=%d", tokens, float64(limit), burst)

	// 检查是否接近限流阈值
	if tokens < float64(burst)*0.2 {
		g.Log().Debugf(ctx, "警告: 限流器令牌数量较低 (%.2f/%d), 接近限流阈值", tokens, burst)
	}

	// 等待令牌
	if err := limiter.Wait(ctx); err != nil {
		return nil, nil, fmt.Errorf("等待限流器超时: %v", err)
	}

	// 发请求
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
		var errorResp APIErrorResponse

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

// 获取交易详情 TODO
func (t *TronAPI) GetTransaction(ctx context.Context, txID string) (Transaction, error) {
	// TODO: 实现获取交易详情的API调用
	return Transaction{}, nil
}

// 获取地址交易历史 TODO
func (t *TronAPI) GetAddressTransactions(ctx context.Context, address string, limit int) ([]Transaction, error) {
	// TODO: 实现获取地址交易历史的API调用
	return nil, nil
}

// 获取地址交易数量 TODO
func (t *TronAPI) GetTransactionCount(ctx context.Context, address, tokenContract string) (int, error) {
	// TODO: 实现获取交易数量的API调用
	return 0, nil
}

// 获取代币余额
func (t *TronAPI) GetTokenBalance(ctx context.Context, address, tokenContract string) (float64, error) {
	// 将Base58地址转换为十六进制地址
	hexAddress, err := utility.Base58ToHexAddress(address)
	if err != nil {
		return 0, fmt.Errorf("地址转换失败: %v", err)
	}

	// 将合约地址转换为十六进制
	hexContractAddress, err := utility.Base58ToHexAddress(tokenContract)
	if err != nil {
		return 0, fmt.Errorf("合约地址转换失败: %v", err)
	}

	// 构造参数
	parameter := fmt.Sprintf("000000000000000000000000%s", hexAddress[2:]) // 去掉41前缀

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
	var response TokenBalanceResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return 0, fmt.Errorf("解析响应失败: %v, 原始响应: %s", err, string(respBody))
	}

	// 检查是否有错误消息
	if response.Message != "" {
		return 0, fmt.Errorf("API返回错误: %s", response.Message)
	}

	// 检查调用是否成功
	if !response.Result.Result {
		// 添加更详细的错误信息
		return 0, fmt.Errorf("合约调用失败, 地址: %s, 合约: %s, 响应: %+v",
			address, tokenContract, response)
	}

	// 解析余额结果
	if len(response.ConstantResult) == 0 {
		return 0, fmt.Errorf("未返回余额结果, 地址: %s, 合约: %s", address, tokenContract)
	}

	// 解析十六进制余额
	balanceHex := response.ConstantResult[0]
	if len(balanceHex) < 2 {
		return 0, fmt.Errorf("余额格式错误: %s", balanceHex)
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

// GetUSDTTransactions 获取地址的USDT交易记录
func (t *TronAPI) GetUSDTTransactions(ctx context.Context, address string, limit int, startTimestamp int64, onlyIncoming bool) ([]TRC20Transaction, error) {
	// 构造URL
	url := fmt.Sprintf("%s/v1/accounts/%s/transactions/trc20", t.BaseURL, address)

	// 查询参数
	query := make(map[string]string)
	query["limit"] = fmt.Sprintf("%d", limit)
	if startTimestamp > 0 {
		query["min_timestamp"] = fmt.Sprintf("%d", startTimestamp)
	}

	// USDT合约地址
	query["contract_address"] = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t"

	// 只查转入
	if onlyIncoming {
		query["only_to"] = "true"
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}

	// 拼接参数
	q := req.URL.Query()
	for k, v := range query {
		q.Add(k, v)
	}
	req.URL.RawQuery = q.Encode()

	// 加API Key
	if t.APIKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", t.APIKey)
	}
	req.Header.Set("Accept", "application/json")

	// 发请求
	_, respBody, err := t.doRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	// 解析响应
	var response TRC20TransactionResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("解析响应失败: %v, 响应: %s", err, string(respBody))
	}

	// 检查成功状态
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
			decimals = 6 // USDT默认6位小数
		}

		divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
		amountFloat := new(big.Float).SetInt(amountBig)
		divisorFloat := new(big.Float).SetInt(divisor)
		result := new(big.Float).Quo(amountFloat, divisorFloat)

		finalAmount, _ := result.Float64()

		// 时间戳转时间
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
	return t.GetUSDTTransactions(ctx, address, limit, startTimestamp, true)
}

// GetIncomingUSDTTransactionsByTimeRange 根据时间范围获取USDT转入交易记录
func (t *TronAPI) GetIncomingUSDTTransactionsByTimeRange(ctx context.Context, address string, startTime, endTime time.Time, limit int) ([]TRC20Transaction, error) {
	// 转毫秒时间戳
	startTimestamp := startTime.UnixNano() / int64(time.Millisecond)

	// 获取转入交易记录
	transactions, err := t.GetUSDTTransactions(ctx, address, limit, startTimestamp, true)
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
	// 转毫秒时间戳
	startTimestamp := startTime.UnixNano() / int64(time.Millisecond)

	// 获取记录
	transactions, err := t.GetUSDTTransactions(ctx, address, limit, startTimestamp, false)
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

// GetLatestBlock 获取波场最新区块
func (t *TronAPI) GetLatestBlock(ctx context.Context) (Block, error) {
	// 请求URL
	url := fmt.Sprintf("%s/wallet/getnowblock", t.BaseURL)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return Block{}, fmt.Errorf("创建请求失败: %v", err)
	}

	// 加API Key
	if t.APIKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", t.APIKey)
	}
	req.Header.Set("Accept", "application/json")

	// 发请求
	_, respBody, err := t.doRequest(ctx, req)
	if err != nil {
		return Block{}, err
	}

	// 解析响应
	var response BlockResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return Block{}, fmt.Errorf("解析响应失败: %v, 响应: %s", err, string(respBody))
	}

	// 提取交易ID
	var txIDs []string
	for _, tx := range response.Transactions {
		txIDs = append(txIDs, tx.TxID)
	}

	// 时间戳转时间
	blockTime := time.Unix(response.BlockHeader.RawData.Timestamp/1000, 0)

	// 构造返回
	block := Block{
		BlockID:        response.BlockID,
		BlockNumber:    response.BlockHeader.RawData.Number,
		Timestamp:      response.BlockHeader.RawData.Timestamp,
		TransactionNum: len(response.Transactions),
		Transactions:   txIDs,
		Confirmed:      true, // 最新区块一般已确认
		BlockTime:      blockTime,
	}

	return block, nil
}
