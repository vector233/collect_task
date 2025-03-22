// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// TOrderAddressRecordResult is the golang structure for table t_order_address_record_result.
type TOrderAddressRecordResult struct {
	Id               int64       `json:"id"               orm:"id"                 description:""`       //
	FromAddressPart  string      `json:"fromAddressPart"  orm:"from_address_part"  description:"任务"`     // 任务
	Address          string      `json:"address"          orm:"address"            description:"结果"`     // 结果
	CreateTime       *gtime.Time `json:"createTime"       orm:"create_time"        description:"任务创建时间"` // 任务创建时间
	PrivateAddress   string      `json:"privateAddress"   orm:"private_address"    description:"结果后缀"`   // 结果后缀
	MatchSuccessTime *gtime.Time `json:"matchSuccessTime" orm:"match_success_time" description:"匹配成功时间"` // 匹配成功时间
}
