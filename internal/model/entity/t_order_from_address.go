// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// TOrderFromAddress is the golang structure for table t_order_from_address.
type TOrderFromAddress struct {
	Id          int64       `json:"id"          orm:"id"           description:"主键"`     // 主键
	FromAddress string      `json:"fromAddress" orm:"from_address" description:"from地址"` // from地址
	LastAmount  float64     `json:"lastAmount"  orm:"last_amount"  description:"最后到账金额"` // 最后到账金额
	LastTime    *gtime.Time `json:"lastTime"    orm:"last_time"    description:"最后到账时间"` // 最后到账时间
	CreateTime  *gtime.Time `json:"createTime"  orm:"create_time"  description:"创建时间"`   // 创建时间
	QueryTime   *gtime.Time `json:"queryTime"   orm:"query_time"   description:"查询时间"`   // 查询时间
	Balance     float64     `json:"balance"     orm:"balance"      description:"余额"`     // 余额
}
