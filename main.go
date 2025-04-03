package main

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/glog"

	_ "tron-lion/internal/packed"

	_ "github.com/gogf/gf/contrib/drivers/mysql/v2"
	_ "github.com/gogf/gf/contrib/nosql/redis/v2"

	"github.com/gogf/gf/v2/os/gctx"

	"tron-lion/internal/cmd"
)

func main() {
	g.Log().SetLevel(glog.LEVEL_INFO)
	cmd.Main.Run(gctx.GetInitCtx())
}
