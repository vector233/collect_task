// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// TOrderFromAddress is the golang structure of table t_order_from_address for DAO operations like Where/Data.
type TOrderFromAddress struct {
	g.Meta      `orm:"table:t_order_from_address, do:true"`
	Id          interface{} // 主键
	FromAddress interface{} // from地址
	LastAmount  interface{} // 最后到账金额
	LastTime    *gtime.Time // 最后到账时间
	CreateTime  *gtime.Time // 创建时间
	QueryTime   *gtime.Time // 查询时间
	Balance     interface{} // 余额
}
