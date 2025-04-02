package cmd

import (
	"context"

	"github.com/gogf/gf/v2/os/gcmd"

	"github.com/bivdex/tron-lion/internal/cmd/analysis" // 添加这一行
	"github.com/bivdex/tron-lion/internal/cmd/generate_data"
)

var (
	Main = gcmd.Command{
		Name:  "main",
		Usage: "main",
		Brief: "start http server",
		Func: func(ctx context.Context, parser *gcmd.Parser) (err error) {
			// s := g.Server()
			// s.Group("/", func(group *ghttp.RouterGroup) {
			// 	group.Middleware(ghttp.MiddlewareHandlerResponse)
			// 	group.Bind(
			// 		hello.NewV1(),
			// 	)
			// })
			// s.Run()
			return nil
		},
	}
)

func init() {
	err := Main.AddCommand(
		&TronPipei,
		&TronLianghao,
		&TronTest,
		&generate_data.TronGenerateA,
		&generate_data.TronGenerateE,
		&generate_data.TronGenerateD,
		&generate_data.TronGenerateC,
		&analysis.TronCron,
		&analysis.TronBalance,
	)
	if err != nil {
		panic(err)
	}
}
