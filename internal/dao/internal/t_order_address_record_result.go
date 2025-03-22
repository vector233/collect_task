// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// TOrderAddressRecordResultDao is the data access object for the table t_order_address_record_result.
type TOrderAddressRecordResultDao struct {
	table    string                           // table is the underlying table name of the DAO.
	group    string                           // group is the database configuration group name of the current DAO.
	columns  TOrderAddressRecordResultColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler               // handlers for customized model modification.
}

// TOrderAddressRecordResultColumns defines and stores column names for the table t_order_address_record_result.
type TOrderAddressRecordResultColumns struct {
	Id               string //
	FromAddressPart  string // 任务
	Address          string // 结果
	CreateTime       string // 任务创建时间
	PrivateAddress   string // 结果后缀
	MatchSuccessTime string // 匹配成功时间
}

// tOrderAddressRecordResultColumns holds the columns for the table t_order_address_record_result.
var tOrderAddressRecordResultColumns = TOrderAddressRecordResultColumns{
	Id:               "id",
	FromAddressPart:  "from_address_part",
	Address:          "address",
	CreateTime:       "create_time",
	PrivateAddress:   "private_address",
	MatchSuccessTime: "match_success_time",
}

// NewTOrderAddressRecordResultDao creates and returns a new DAO object for table data access.
func NewTOrderAddressRecordResultDao(handlers ...gdb.ModelHandler) *TOrderAddressRecordResultDao {
	return &TOrderAddressRecordResultDao{
		group:    "default",
		table:    "t_order_address_record_result",
		columns:  tOrderAddressRecordResultColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *TOrderAddressRecordResultDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *TOrderAddressRecordResultDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *TOrderAddressRecordResultDao) Columns() TOrderAddressRecordResultColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *TOrderAddressRecordResultDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *TOrderAddressRecordResultDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *TOrderAddressRecordResultDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
