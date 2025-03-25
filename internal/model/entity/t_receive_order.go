// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// TReceiveOrder is the golang structure for table t_receive_order.
type TReceiveOrder struct {
	Id              int64       `json:"id"              orm:"id"                description:""`                         //
	OrderNo         string      `json:"orderNo"         orm:"order_no"          description:"订单号"`                      // 订单号
	FromAddressPart string      `json:"fromAddressPart" orm:"from_address_part" description:"前3后4码"`                    // 前3后4码
	ToAddress       string      `json:"toAddress"       orm:"to_address"        description:"目标地址"`                     // 目标地址
	Amount          float64     `json:"amount"          orm:"amount"            description:"数量"`                       // 数量
	OrderTime       string      `json:"orderTime"       orm:"order_time"        description:"订单时间"`                     // 订单时间
	CreateTime      *gtime.Time `json:"createTime"      orm:"create_time"       description:"创建时间"`                     // 创建时间
	Initialization  int         `json:"initialization"  orm:"initialization"    description:"是否初始化:0未初始化 1.已初始化"`       // 是否初始化:0未初始化 1.已初始化
	ErrorData       int         `json:"errorData"       orm:"error_data"        description:"异常数据:1正常.2.重复 3.金额大于1000"` // 异常数据:1正常.2.重复 3.金额大于1000
	WaitMatch       int         `json:"waitMatch"       orm:"wait_match"        description:"是否匹配:0.等待匹配 1.已匹配"`        // 是否匹配:0.等待匹配 1.已匹配
	IsDel           string      `json:"isDel"           orm:"is_del"            description:"是否删除 0无，1已删"`              // 是否删除 0无，1已删
}
