// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// TReceiveOrder is the golang structure of table t_receive_order for DAO operations like Where/Data.
type TReceiveOrder struct {
	g.Meta          `orm:"table:t_receive_order, do:true"`
	Id              interface{} //
	OrderNo         interface{} // 订单号
	FromAddressPart interface{} // 前3后4码
	ToAddress       interface{} // 目标地址
	Amount          interface{} // 数量
	OrderTime       interface{} // 订单时间
	CreateTime      *gtime.Time // 创建时间
	Initialization  interface{} // 是否初始化:0未初始化 1.已初始化
	ErrorData       interface{} // 异常数据:1正常.2.重复 3.金额大于1000
	WaitMatch       interface{} // 是否匹配:0.等待匹配 1.已匹配
	IsDel           interface{} // 是否删除 0无，1已删
}
