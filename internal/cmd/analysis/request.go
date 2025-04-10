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
// func (t *TronAPI) GetTransaction(ctx context.Context, txID string) (Transaction, error) {
// 	// TODO: 实现获取交易详情的API调用
// 	return Transaction{}, nil
// }

// 获取地址交易历史 TODO
func (t *TronAPI) GetAddressTransactions(ctx context.Context, address string, limit int) ([]Transaction, error) {
	// TODO: 实现获取地址交易历史的API调用
	return nil, nil
}

// 获取地址交易数量
func (t *TronAPI) GetTransactionCount(ctx context.Context, params TransactionCountParams) (int, error) {
	// 构造URL - 使用Tronscan API
	url := fmt.Sprintf("https://apilist.tronscan.org/api/token_trc20/transfers")

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("创建请求失败: %v", err)
	}

	// 添加查询参数
	q := req.URL.Query()
	q.Add("count", "true") // 请求返回总数

	// 设置起始位置
	if params.Start > 0 {
		q.Add("start", fmt.Sprintf("%d", params.Start))
	}

	// 设置每页数量
	if params.Limit <= 0 {
		params.Limit = 1 // 只需要获取数量，不需要实际数据
	}
	q.Add("limit", fmt.Sprintf("%d", params.Limit))

	// 设置合约地址
	if params.ContractAddress != "" {
		q.Add("contract_address", params.ContractAddress)
	}

	// 设置开始时间
	if params.StartTimestamp != nil {
		startTs := params.StartTimestamp.UnixNano() / int64(time.Millisecond)
		q.Add("start_timestamp", fmt.Sprintf("%d", startTs))
	}

	// 设置结束时间
	if params.EndTimestamp != nil {
		endTs := params.EndTimestamp.UnixNano() / int64(time.Millisecond)
		q.Add("end_timestamp", fmt.Sprintf("%d", endTs))
	}

	// 设置是否只返回已确认的交易
	if params.Confirm != nil {
		if *params.Confirm {
			q.Add("confirm", "true")
		} else {
			q.Add("confirm", "false")
		}
	}

	// 设置相关地址
	if params.RelatedAddress != "" {
		q.Add("relatedAddress", params.RelatedAddress)
	}

	// 设置发送方地址
	if params.FromAddress != "" {
		q.Add("fromAddress", params.FromAddress)
	}

	// 设置接收方地址
	if params.ToAddress != "" {
		q.Add("toAddress", params.ToAddress)
	}

	req.URL.RawQuery = q.Encode()

	// 设置请求头
	req.Header.Set("Accept", "application/json")

	// 设置API Key
	if t.APIKey != "" {
		// Tronscan API使用APIKEY作为请求头
		req.Header.Set("APIKEY", t.APIKey)

		// 同时保留TRON-PRO-API-KEY以兼容TronGrid API
		req.Header.Set("TRON-PRO-API-KEY", t.APIKey)

		g.Log().Debugf(ctx, "使用API Key [%s] 请求Tronscan API", maskAPIKey(t.APIKey))
	}

	// 发送请求
	_, respBody, err := t.doRequest(ctx, req)
	if err != nil {
		return 0, fmt.Errorf("请求交易数量失败: %v", err)
	}

	// 解析响应
	var response struct {
		Total      int `json:"total"`
		RangeTotal int `json:"rangeTotal"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return 0, fmt.Errorf("解析响应失败: %v, 原始响应: %s", err, string(respBody))
	}

	g.Log().Debugf(ctx, "地址 %s 的交易数量: %d, %d", params.RelatedAddress, response.Total, response.RangeTotal)
	return response.Total, nil
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
func (t *TronAPI) GetUSDTTransactions(ctx context.Context, address string, params TRC20TransactionParams) (TRC20TransactionResponse, error) {
	// 构造URL
	url := fmt.Sprintf("%s/v1/accounts/%s/transactions/trc20", t.BaseURL, address)

	// 查询参数
	query := make(map[string]string)

	// 设置限制数量
	if params.Limit <= 0 {
		params.Limit = 20 // 默认值
	} else if params.Limit > 200 {
		params.Limit = 200 // 最大值
	}
	query["limit"] = fmt.Sprintf("%d", params.Limit)

	// 设置指纹
	if params.Fingerprint != "" {
		query["fingerprint"] = params.Fingerprint
	}

	// 设置排序方式
	if params.OrderBy != "" {
		query["order_by"] = params.OrderBy
	}

	// 设置最小时间戳
	if params.MinTimestamp != nil {
		minTimestamp := params.MinTimestamp.UnixNano() / int64(time.Millisecond)
		query["min_timestamp"] = fmt.Sprintf("%d", minTimestamp)
	}

	// 设置最大时间戳
	if params.MaxTimestamp != nil {
		maxTimestamp := params.MaxTimestamp.UnixNano() / int64(time.Millisecond)
		query["max_timestamp"] = fmt.Sprintf("%d", maxTimestamp)
	}

	// 设置合约地址，默认为USDT
	if params.ContractAddress != "" {
		query["contract_address"] = params.ContractAddress
	} else {
		query["contract_address"] = "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t" // 波场USDT合约地址
	}

	// 设置只查询已确认交易
	if params.OnlyConfirmed != nil && *params.OnlyConfirmed {
		query["only_confirmed"] = "true"
	}

	// 设置只查询未确认交易
	if params.OnlyUnconfirmed != nil && *params.OnlyUnconfirmed {
		query["only_unconfirmed"] = "true"
	}

	// 设置只查询转入交易
	if params.OnlyTo != nil && *params.OnlyTo {
		query["only_to"] = "true"
	}

	// 设置只查询转出交易
	if params.OnlyFrom != nil && *params.OnlyFrom {
		query["only_from"] = "true"
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return TRC20TransactionResponse{}, fmt.Errorf("创建请求失败: %v", err)
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
		return TRC20TransactionResponse{}, err
	}

	// 解析响应
	var response TRC20TransactionResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return TRC20TransactionResponse{}, fmt.Errorf("解析响应失败: %v, 响应: %s", err, string(respBody))
	}

	// 检查成功状态
	if !response.Success {
		return response, fmt.Errorf("API调用失败")
	}

	return response, nil
}

// ConvertToTRC20Transactions 将API响应转换为TRC20Transaction结构
func ConvertToTRC20Transactions(response TRC20TransactionResponse) ([]TRC20Transaction, error) {
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
			Timestamp:      timestamp,
		})
	}

	return transactions, nil
}

// GetIncomingUSDTTransactions 获取地址的USDT转入交易记录
func (t *TronAPI) GetIncomingUSDTTransactions(ctx context.Context, address string, limit int, startTimestamp int64) ([]TRC20Transaction, error) {
	// 创建参数
	onlyTo := true
	params := TRC20TransactionParams{
		Limit:  limit,
		OnlyTo: &onlyTo,
	}

	// 设置最小时间戳
	if startTimestamp > 0 {
		minTime := time.Unix(startTimestamp/1000, 0)
		params.MinTimestamp = &minTime
	}

	// 获取原始响应
	response, err := t.GetUSDTTransactions(ctx, address, params)
	if err != nil {
		return nil, err
	}

	// 转换为TRC20Transaction结构
	return ConvertToTRC20Transactions(response)
}

// GetIncomingUSDTTransactionsByTimeRange 根据时间范围获取USDT转入交易记录
func (t *TronAPI) GetIncomingUSDTTransactionsByTimeRange(ctx context.Context, address string, params TRC20TransactionParams) ([]TRC20Transaction, error) {
	// 确保设置了只查询转入交易
	onlyTo := true
	params.OnlyTo = &onlyTo

	// 确保设置了合理的限制
	if params.Limit <= 0 {
		params.Limit = 20 // 默认值
	}

	// 获取原始响应
	response, err := t.GetUSDTTransactions(ctx, address, params)
	if err != nil {
		return nil, err
	}

	// 转换为TRC20Transaction结构
	return ConvertToTRC20Transactions(response)
}

// GetUSDTTransactionsByTimeRange 根据时间范围获取USDT交易记录
func (t *TronAPI) GetUSDTTransactionsByTimeRange(ctx context.Context, address string, params TRC20TransactionParams, limit int) ([]TRC20Transaction, error) {
	// 如果limit小于等于0，使用默认值200
	if limit <= 0 {
		limit = 200
	}

	// 存储所有交易记录
	var allTransactions []TRC20Transaction

	// 每次请求的数量，不超过API限制
	requestLimit := 200
	if limit < requestLimit {
		requestLimit = limit
	}

	// 当前已获取的记录数
	var fetchedCount int

	// 当前指纹，用于分页
	var fingerprint string

	// 循环获取交易记录，直到达到limit或没有更多数据
	for fetchedCount < limit {
		// 设置本次请求的参数
		currentParams := params
		currentParams.Limit = requestLimit // 每次请求使用相同的limit值

		// 如果有指纹，设置指纹用于分页
		if fingerprint != "" {
			currentParams.Fingerprint = fingerprint
		}

		// 获取原始响应
		response, err := t.GetUSDTTransactions(ctx, address, currentParams)
		if err != nil {
			return nil, fmt.Errorf("获取交易记录失败: %w", err)
		}

		// 转换为TRC20Transaction结构
		transactions, err := ConvertToTRC20Transactions(response)
		if err != nil {
			return nil, fmt.Errorf("转换交易记录失败: %w", err)
		}

		// 添加到结果集，但不超过总的limit限制
		remainingNeeded := limit - fetchedCount
		if len(transactions) > remainingNeeded {
			allTransactions = append(allTransactions, transactions[:remainingNeeded]...)
			fetchedCount += remainingNeeded
		} else {
			allTransactions = append(allTransactions, transactions...)
			fetchedCount += len(transactions)
		}

		// 检查是否有更多数据
		if response.Meta.Fingerprint == "" || len(transactions) < requestLimit {
			// 没有更多数据了
			break
		}

		// 检查是否已经达到请求的限制
		if fetchedCount >= limit {
			break
		}

		// 更新指纹，用于下一次请求
		fingerprint = response.Meta.Fingerprint

		// 记录日志
		g.Log().Debugf(ctx, "已获取 %d/%d 条交易记录，继续获取下一页...", fetchedCount, limit)

		//select {
		//case <-ctx.Done():
		//	return nil, ctx.Err()
		//case <-time.After(100 * time.Millisecond):
		//	// 100ms 继续执行
		//}
	}

	return allTransactions, nil
}

// GetLatestBlock 获取波场最新区块
func (t *TronAPI) GetLatestBlock(ctx context.Context) (BlockResponse, error) {
	// 请求URL
	url := fmt.Sprintf("%s/wallet/getnowblock", t.BaseURL)

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return BlockResponse{}, fmt.Errorf("创建请求失败: %v", err)
	}

	// 加API Key
	if t.APIKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", t.APIKey)
	}
	req.Header.Set("Accept", "application/json")

	// 发请求
	_, respBody, err := t.doRequest(ctx, req)
	if err != nil {
		return BlockResponse{}, err
	}

	// 解析响应
	var response BlockResponse
	if err := json.Unmarshal(respBody, &response); err != nil {
		return BlockResponse{}, fmt.Errorf("解析响应失败: %v, 响应: %s", err, string(respBody))
	}

	return response, nil
}

// 获取交易详情并判断是否为USDT交易
func (t *TronAPI) GetTransaction(ctx context.Context, txID string) (Transaction, error) {
	// 获取交易基本信息
	txResponse, err := t.fetchTransactionBasicInfo(ctx, txID)
	if err != nil {
		return Transaction{}, err
	}

	// 获取交易详细信息
	txInfoResponse, err := t.fetchTransactionDetailInfo(ctx, txID)
	if err != nil {
		return Transaction{}, err
	}
	fmt.Println(txResponse)
	fmt.Println(txInfoResponse)

	return Transaction{}, nil
}

// 获取交易基本信息
func (t *TronAPI) fetchTransactionBasicInfo(ctx context.Context, txID string) (BlockTransaction, error) {
	// 构造URL
	url := fmt.Sprintf("%s/wallet/gettransactionbyid", t.BaseURL)

	// 构建请求体
	requestBody := map[string]string{
		"value": txID,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return BlockTransaction{}, fmt.Errorf("构建请求体失败: %v", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestJSON)))
	if err != nil {
		return BlockTransaction{}, fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if t.APIKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", t.APIKey)
	}

	// 发送请求
	_, respBody, err := t.doRequest(ctx, req)
	if err != nil {
		return BlockTransaction{}, err
	}

	// 解析响应
	var txResponse BlockTransaction
	if err := json.Unmarshal(respBody, &txResponse); err != nil {
		return BlockTransaction{}, fmt.Errorf("解析交易响应失败: %v, 响应: %s", err, string(respBody))
	}

	return txResponse, nil
}

// 获取交易详细信息
func (t *TronAPI) fetchTransactionDetailInfo(ctx context.Context, txID string) (TransactionInfoResponse, error) {
	// 构造URL
	url := fmt.Sprintf("%s/wallet/gettransactioninfobyid", t.BaseURL)

	// 构建请求体
	requestBody := map[string]string{
		"value": txID,
	}

	requestJSON, err := json.Marshal(requestBody)
	if err != nil {
		return TransactionInfoResponse{}, fmt.Errorf("构建请求体失败: %v", err)
	}

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(requestJSON)))
	if err != nil {
		return TransactionInfoResponse{}, fmt.Errorf("创建交易信息请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if t.APIKey != "" {
		req.Header.Set("TRON-PRO-API-KEY", t.APIKey)
	}

	// 发送请求
	_, respBody, err := t.doRequest(ctx, req)
	if err != nil {
		return TransactionInfoResponse{}, err
	}

	// 解析交易信息响应
	var txInfoResponse TransactionInfoResponse
	if err := json.Unmarshal(respBody, &txInfoResponse); err != nil {
		return TransactionInfoResponse{}, fmt.Errorf("解析交易信息响应失败: %v, 响应: %s", err, string(respBody))
	}

	return txInfoResponse, nil
}
