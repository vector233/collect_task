// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// TOrderAddressRecordResult is the golang structure of table t_order_address_record_result for DAO operations like Where/Data.
type TOrderAddressRecordResult struct {
	g.Meta           `orm:"table:t_order_address_record_result, do:true"`
	Id               interface{} //
	FromAddressPart  interface{} // 任务
	Address          interface{} // 结果
	CreateTime       *gtime.Time // 任务创建时间
	PrivateAddress   interface{} // 结果后缀
	MatchSuccessTime *gtime.Time // 匹配成功时间
}
