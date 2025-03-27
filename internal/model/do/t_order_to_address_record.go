// =================================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// =================================================================================

package do

import (
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gtime"
)

// TOrderToAddressRecord is the golang structure of table t_order_to_address_record for DAO operations like Where/Data.
type TOrderToAddressRecord struct {
	g.Meta          `orm:"table:t_order_to_address_record, do:true"`
	Id              interface{} //
	FromAddressPart interface{} // 前三后四码
	ToAddress       interface{} // 目标地址
	CreateTime      *gtime.Time // 创建时间
}
