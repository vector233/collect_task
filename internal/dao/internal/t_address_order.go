// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// TAddressOrderDao is the data access object for the table t_address_order.
type TAddressOrderDao struct {
	table    string               // table is the underlying table name of the DAO.
	group    string               // group is the database configuration group name of the current DAO.
	columns  TAddressOrderColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler   // handlers for customized model modification.
}

// TAddressOrderColumns defines and stores column names for the table t_address_order.
type TAddressOrderColumns struct {
	Id               string // 主键
	Address          string // 地址
	Balance          string // 余额
	QueryTime        string // 查询时间
	OrderAmount      string // 订单金额
	IncomeTime       string // 进账时间
	IncomeCreateTime string // 监听时间
}

// tAddressOrderColumns holds the columns for the table t_address_order.
var tAddressOrderColumns = TAddressOrderColumns{
	Id:               "id",
	Address:          "address",
	Balance:          "balance",
	QueryTime:        "query_time",
	OrderAmount:      "order_amount",
	IncomeTime:       "income_time",
	IncomeCreateTime: "income_create_time",
}

// NewTAddressOrderDao creates and returns a new DAO object for table data access.
func NewTAddressOrderDao(handlers ...gdb.ModelHandler) *TAddressOrderDao {
	return &TAddressOrderDao{
		group:    "default",
		table:    "t_address_order",
		columns:  tAddressOrderColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *TAddressOrderDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *TAddressOrderDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *TAddressOrderDao) Columns() TAddressOrderColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *TAddressOrderDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *TAddressOrderDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *TAddressOrderDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
