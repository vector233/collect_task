// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// TOrderToAddressRecordDao is the data access object for the table t_order_to_address_record.
type TOrderToAddressRecordDao struct {
	table    string                       // table is the underlying table name of the DAO.
	group    string                       // group is the database configuration group name of the current DAO.
	columns  TOrderToAddressRecordColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler           // handlers for customized model modification.
}

// TOrderToAddressRecordColumns defines and stores column names for the table t_order_to_address_record.
type TOrderToAddressRecordColumns struct {
	Id              string //
	FromAddressPart string // 前三后四码
	ToAddress       string // 目标地址
	CreateTime      string // 创建时间
}

// tOrderToAddressRecordColumns holds the columns for the table t_order_to_address_record.
var tOrderToAddressRecordColumns = TOrderToAddressRecordColumns{
	Id:              "id",
	FromAddressPart: "from_address_part",
	ToAddress:       "to_address",
	CreateTime:      "create_time",
}

// NewTOrderToAddressRecordDao creates and returns a new DAO object for table data access.
func NewTOrderToAddressRecordDao(handlers ...gdb.ModelHandler) *TOrderToAddressRecordDao {
	return &TOrderToAddressRecordDao{
		group:    "default",
		table:    "t_order_to_address_record",
		columns:  tOrderToAddressRecordColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *TOrderToAddressRecordDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *TOrderToAddressRecordDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *TOrderToAddressRecordDao) Columns() TOrderToAddressRecordColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *TOrderToAddressRecordDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *TOrderToAddressRecordDao) Ctx(ctx context.Context) *gdb.Model {
	model := dao.DB().Model(dao.table)
	for _, handler := range dao.handlers {
		model = handler(model)
	}
	return model.Safe().Ctx(ctx)
}

// Transaction wraps the transaction logic using function f.
// It rolls back the transaction and returns the error if function f returns a non-nil error.
// It commits the transaction and returns nil if function f returns nil.
//
// Note: Do not commit or roll back the transaction in function f,
// as it is automatically handled by this function.
func (dao *TOrderToAddressRecordDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
