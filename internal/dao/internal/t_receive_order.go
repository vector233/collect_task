// ==========================================================================
// Code generated and maintained by GoFrame CLI tool. DO NOT EDIT.
// ==========================================================================

package internal

import (
	"context"

	"github.com/gogf/gf/v2/database/gdb"
	"github.com/gogf/gf/v2/frame/g"
)

// TReceiveOrderDao is the data access object for the table t_receive_order.
type TReceiveOrderDao struct {
	table    string               // table is the underlying table name of the DAO.
	group    string               // group is the database configuration group name of the current DAO.
	columns  TReceiveOrderColumns // columns contains all the column names of Table for convenient usage.
	handlers []gdb.ModelHandler   // handlers for customized model modification.
}

// TReceiveOrderColumns defines and stores column names for the table t_receive_order.
type TReceiveOrderColumns struct {
	Id              string //
	OrderNo         string // 订单号
	FromAddressPart string // 前3后4码
	ToAddress       string // 目标地址
	Amount          string // 数量
	OrderTime       string // 订单时间
	CreateTime      string // 创建时间
	Initialization  string // 是否初始化:0未初始化 1.已初始化
	ErrorData       string // 异常数据:1正常.2.重复 3.金额大于1000
	WaitMatch       string // 是否匹配:0.等待匹配 1.已匹配
	IsDel           string // 是否删除 0无，1已删
}

// tReceiveOrderColumns holds the columns for the table t_receive_order.
var tReceiveOrderColumns = TReceiveOrderColumns{
	Id:              "id",
	OrderNo:         "order_no",
	FromAddressPart: "from_address_part",
	ToAddress:       "to_address",
	Amount:          "amount",
	OrderTime:       "order_time",
	CreateTime:      "create_time",
	Initialization:  "initialization",
	ErrorData:       "error_data",
	WaitMatch:       "wait_match",
	IsDel:           "is_del",
}

// NewTReceiveOrderDao creates and returns a new DAO object for table data access.
func NewTReceiveOrderDao(handlers ...gdb.ModelHandler) *TReceiveOrderDao {
	return &TReceiveOrderDao{
		group:    "default",
		table:    "t_receive_order",
		columns:  tReceiveOrderColumns,
		handlers: handlers,
	}
}

// DB retrieves and returns the underlying raw database management object of the current DAO.
func (dao *TReceiveOrderDao) DB() gdb.DB {
	return g.DB(dao.group)
}

// Table returns the table name of the current DAO.
func (dao *TReceiveOrderDao) Table() string {
	return dao.table
}

// Columns returns all column names of the current DAO.
func (dao *TReceiveOrderDao) Columns() TReceiveOrderColumns {
	return dao.columns
}

// Group returns the database configuration group name of the current DAO.
func (dao *TReceiveOrderDao) Group() string {
	return dao.group
}

// Ctx creates and returns a Model for the current DAO. It automatically sets the context for the current operation.
func (dao *TReceiveOrderDao) Ctx(ctx context.Context) *gdb.Model {
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
func (dao *TReceiveOrderDao) Transaction(ctx context.Context, f func(ctx context.Context, tx gdb.TX) error) (err error) {
	return dao.Ctx(ctx).Transaction(ctx, f)
}
