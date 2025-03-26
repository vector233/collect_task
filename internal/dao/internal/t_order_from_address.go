// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// TOrderFromAddressDao is the data access object for the table t_order_from_address.
type TOrderFromAddressDao struct {
	table    string                   // table is the underlying table name of the DAO.
	group    string                   // group is the database configuration group name of the current DAO.
	columns  TOrderFromAddressColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler       // handlers for customized model modification.
}

// TOrderFromAddressColumns defines and stores column names for the table t_order_from_address.
type TOrderFromAddressColumns struct {
	Id          string // 主键
	FromAddress string // from地址
	CreateTime  string // 创建时间
}

// tOrderFromAddressColumns holds the columns for the table t_order_from_address.
var tOrderFromAddressColumns = TOrderFromAddressColumns{
	Id:          "id",
	FromAddress: "from_address",
	CreateTime:  "create_time",
}

// NewTOrderFromAddressDao creates and returns a new DAO object for table data access.
func NewTOrderFromAddressDao(handlers ...gdb.ModelHandler) *TOrderFromAddressDao {
	return &TOrderFromAddressDao{
		group:    "default",
		table:    "t_order_from_address",
		columns:  tOrderFromAddressColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *TOrderFromAddressDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *TOrderFromAddressDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *TOrderFromAddressDao) Columns() TOrderFromAddressColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *TOrderFromAddressDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *TOrderFromAddressDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *TOrderFromAddressDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
