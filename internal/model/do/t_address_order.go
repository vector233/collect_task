// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// TAddressOrder is the golang structure of table t_address_order for DAO operations like Where/Data.
type TAddressOrder struct {
	g.Meta           `orm:"table:t_address_order, do:true"`
	Id               interface{} // 主键
	Address          interface{} // 地址
	Balance          interface{} // 余额
	QueryTime        *gtime.Time // 查询时间
	OrderAmount      interface{} // 订单金额
	IncomeTime       *gtime.Time // 进账时间
	IncomeCreateTime *gtime.Time // 监听时间
}
