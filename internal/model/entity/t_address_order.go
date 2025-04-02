// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// TAddressOrder is the golang structure for table t_address_order.
type TAddressOrder struct {
	Id               int64       `json:"id"               orm:"id"                 description:"主键"`   // 主键
	Address          string      `json:"address"          orm:"address"            description:"地址"`   // 地址
	Balance          float64     `json:"balance"          orm:"balance"            description:"余额"`   // 余额
	QueryTime        *gtime.Time `json:"queryTime"        orm:"query_time"         description:"查询时间"` // 查询时间
	OrderAmount      float64     `json:"orderAmount"      orm:"order_amount"       description:"订单金额"` // 订单金额
	IncomeTime       *gtime.Time `json:"incomeTime"       orm:"income_time"        description:"进账时间"` // 进账时间
	IncomeCreateTime *gtime.Time `json:"incomeCreateTime" orm:"income_create_time" description:"监听时间"` // 监听时间
}
