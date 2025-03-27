// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package entity

import (
	"github.com/gogf/gf/v2/os/gtime"
)

// TOrderToAddressRecord is the golang structure for table t_order_to_address_record.
type TOrderToAddressRecord struct {
	Id              int64       `json:"id"              orm:"id"                description:""`      //
	FromAddressPart string      `json:"fromAddressPart" orm:"from_address_part" description:"前三后四码"` // 前三后四码
	ToAddress       string      `json:"toAddress"       orm:"to_address"        description:"目标地址"`  // 目标地址
	CreateTime      *gtime.Time `json:"createTime"      orm:"create_time"       description:"创建时间"`  // 创建时间
}
