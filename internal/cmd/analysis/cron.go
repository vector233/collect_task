package analysis

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gcmd"
)

var (
	TronCron = gcmd.Command{
		Name:  "cron",
		Usage: "tron cron",
		Brief: "启动定时任务",
		Func:  runCron,
	}

	TronBalance = gcmd.Command{
		Name:  "balance",
		Usage: "tron balance",
		Brief: "启动余额采集",
		Func:  UpdateBalance,
	}
)

// 初始化API和监控器
func initAPIAndMonitor(ctx context.Context) (*TronAPI, *BalanceMonitor) {
	// 读取配置
	baseURL := g.Cfg().MustGet(ctx, "tron.api.baseURL", "http://104.233.192.15:8090").String()
	apiKey := g.Cfg().MustGet(ctx, "tron.api.key", "").String()
	timeoutSeconds := g.Cfg().MustGet(ctx, "tron.api.timeout", 30).Int()
	usdtContract := g.Cfg().MustGet(ctx, "tron.api.usdt.contract", "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t").String()

	// 获取限流配置
	requestsPerSecond := g.Cfg().MustGet(ctx, "tron.api.rateLimit.requestsPerSecond", 15).Int()
	rateLimitBucket := g.Cfg().MustGet(ctx, "tron.api.rateLimit.bucket", 10).Int()

	// 创建TRON API客户端
	tronAPI := NewTronAPI(baseURL, apiKey)
	tronAPI.HttpTimeout = time.Duration(timeoutSeconds) * time.Second

	// 设置限流参数
	tronAPI.SetRateLimit(requestsPerSecond, rateLimitBucket)
	g.Log().Infof(ctx, "API请求限流设置为: %d 请求/秒, 桶容量: %d", requestsPerSecond, rateLimitBucket)
	g.Log().Infof(ctx, "API请求超时时间设置为: %d秒", timeoutSeconds)

	// 创建余额监控器
	monitor := NewBalanceMonitor(tronAPI, usdtContract)

	// 设置表名
	tableName := g.Cfg().MustGet(ctx, "tron.balance.table", "t_order_from_address").String()
	monitor.SetTable(tableName)

	// 设置并发数
	concurrency := g.Cfg().MustGet(ctx, "tron.balance.concurrency", 15).Int()
	monitor.SetConcurrency(concurrency)

	// 设置批处理大小
	batchSize := g.Cfg().MustGet(ctx, "tron.balance.batchSize", 50).Int()
	monitor.SetBatchSize(batchSize)

	return tronAPI, monitor
}

func runCron(ctx context.Context, parser *gcmd.Parser) error {
	// 初始化API和监控器
	_, monitor := initAPIAndMonitor(ctx)

	// 从配置中获取定时任务执行周期，默认每30分钟执行一次
	cronPattern := g.Cfg().MustGet(ctx, "tron.balance.cron", "0 */30 * * * *").String()

	// 创建信号通道，监听终止信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动监控
	err := monitor.StartMonitor(cronPattern)
	if err != nil {
		g.Log().Fatalf(ctx, "启动余额监控失败: %v", err)
	}

	g.Log().Info(ctx, "余额监控初始化完成")

	// 等待终止信号
	sig := <-sigChan
	g.Log().Infof(ctx, "接收到信号 %v，准备退出程序", sig)

	g.Log().Info(ctx, "正在等待任务完成...")
	time.Sleep(2 * time.Second)

	return nil
}

// 手动触发余额更新
func UpdateBalance(ctx context.Context, parser *gcmd.Parser) error {
	// 初始化API和监控器
	_, monitor := initAPIAndMonitor(ctx)

	// 执行余额更新
	monitor.UpdateAllAddressesBalance(ctx)
	return nil
}
