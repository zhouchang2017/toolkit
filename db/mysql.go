package db

import (
	"context"
	"database/sql"
	"errors"
	"github.com/didi/gendry/builder"
	"github.com/didi/gendry/scanner"
	"github.com/zhouchang2017/toolkit/config/mysqlconfig"
)

var (
	_ dbExecutor = &sql.DB{}
	_ dbExecutor = &sql.Tx{}
)

type dbExecutor interface {
	Prepare(query string) (*sql.Stmt, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	Exec(query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryRow(query string, args ...interface{}) *sql.Row
}

type SQLDao struct {
	tableName     string
	handleName    string
	selectOrder   string
	defaultSelect []string
	pageSize      int64
	pk            string
}

type SQLDaoOption struct {
	SelectOrder string
	Select      []string
	PageSize    int64
}

func NewSQLDao(tableName string, handleName string, pk string, opt *SQLDaoOption) *SQLDao {
	s := &SQLDao{tableName: tableName, handleName: handleName, pk: pk}
	if opt != nil {
		if opt.SelectOrder != "" {
			s.selectOrder = opt.SelectOrder
		}
		if len(opt.Select) > 0 {
			s.defaultSelect = opt.Select
		} else {
			s.defaultSelect = []string{"*"}
		}

		if opt.PageSize <= 0 {
			s.pageSize = 15
		} else if opt.PageSize > 0 {
			s.pageSize = opt.PageSize
		}
	}
	return s
}

// get mysql databases handler
func (s SQLDao) GetDbHandler() (db *sql.DB, err error) {
	return mysqlconfig.Get(s.handleName)
}

// make query by key conditions
func (s SQLDao) pkCond(key interface{}) map[string]interface{} {
	return map[string]interface{}{s.pk: key}
}

// get one record from table s.tableName by key
func (s SQLDao) findByKey(ctx context.Context, db dbExecutor, key interface{}, record interface{}) (err error) {
	return s.first(ctx, db, s.pkCond(key), record)
}

// get first record from table s.tableName by condition "where"
func (s SQLDao) first(ctx context.Context, db dbExecutor, where map[string]interface{}, record interface{}) (err error) {
	if nil == db {
		return errors.New("sql.DB object couldn't be nil")
	}
	if where == nil {
		where = map[string]interface{}{}
	}
	where["_limit"] = []uint{0, 1}

	if s.selectOrder != "" {
		where["_order"] = s.selectOrder
	}

	cond, vals, err := builder.BuildSelect(s.tableName, where, s.defaultSelect)
	if nil != err {
		return err
	}
	row, err := db.QueryContext(ctx, cond, vals...)
	if nil != err || nil == row {
		return err
	}
	defer row.Close()
	err = scanner.Scan(row, &record)
	return err
}

// gets multiple records from table COLUMNS by condition "where"
// limit < 0 , query all
func (s SQLDao) find(ctx context.Context, db dbExecutor, where map[string]interface{}, offset uint, limit uint, records interface{}) (err error) {
	if nil == db {
		return errors.New("sql.DB object couldn't be nil")
	}
	if where == nil {
		where = map[string]interface{}{}
	}

	if s.selectOrder != "" {
		where["_order"] = s.selectOrder
	}

	if limit == 0 {
		limit = uint(s.pageSize)
	}
	if limit > 0 {
		where["_limit"] = []uint{offset, limit}
	}

	cond, vals, err := builder.BuildSelect(s.tableName, where, s.defaultSelect)
	if nil != err {
		return err
	}
	row, err := db.QueryContext(ctx, cond, vals...)
	if nil != err || nil == row {
		return err
	}
	defer row.Close()
	err = scanner.Scan(row, &records)
	return err
}

// inserts an array of data into table s.tableName
func (s SQLDao) insert(ctx context.Context, db dbExecutor, data ...map[string]interface{}) (id int64, err error) {
	if nil == db {
		return id, errors.New("sql.DB object couldn't be nil")
	}
	cond, vals, err := builder.BuildInsert(s.tableName, data)
	if nil != err {
		return 0, err
	}
	result, err := db.ExecContext(ctx, cond, vals...)
	if nil != err || nil == result {
		return 0, err
	}
	return result.LastInsertId()
}

// updates the table s.tableName
func (s SQLDao) updateByKey(ctx context.Context, db dbExecutor, key interface{}, data map[string]interface{}) (int64, error) {
	return s.update(ctx, db, s.pkCond(key), data)
}

// updates the table s.tableName
func (s SQLDao) update(ctx context.Context, db dbExecutor, where, data map[string]interface{}) (int64, error) {
	if nil == db {
		return 0, errors.New("sql.DB object couldn't be nil")
	}
	cond, vals, err := builder.BuildUpdate(s.tableName, where, data)
	if nil != err {
		return 0, err
	}
	result, err := db.ExecContext(ctx, cond, vals...)
	if nil != err {
		return 0, err
	}
	return result.RowsAffected()
}

// deletes matched records in key
func (s SQLDao) deleteByKey(ctx context.Context, db dbExecutor, key interface{}) (int64, error) {
	return s.delete(ctx, db, s.pkCond(key))
}

// deletes matched records in condition "where"
func (s SQLDao) delete(ctx context.Context, db dbExecutor, where map[string]interface{}) (int64, error) {
	if nil == db {
		return 0, errors.New("sql.DB object couldn't be nil")
	}
	cond, vals, err := builder.BuildDelete(s.tableName, where)
	if nil != err {
		return 0, err
	}
	result, err := db.ExecContext(ctx, cond, vals...)
	if nil != err {
		return 0, err
	}
	return result.RowsAffected()
}

// count matched records in condition "where"
func (s SQLDao) Count(ctx context.Context, where map[string]interface{}) (count int64, err error) {
	handler, err := s.GetDbHandler()
	if err != nil {
		return count, err
	}
	selectFields := []string{"count(*)"}

	cond, vals, err := builder.BuildSelect(s.tableName, where, selectFields)
	if err != nil {
		return count, err
	}

	rows, err := handler.Query(cond, vals...)
	if err != nil {
		return count, err
	}
	defer rows.Close()

	count = 0
	if rows.Next() {
		err = rows.Scan(&count)
		if err != nil {
			return count, err
		}
	}
	return count, nil
}

// 通过Key查询
func (s SQLDao) FindByKey(ctx context.Context, key interface{}, record interface{}) (err error) {
	handler, err := s.GetDbHandler()
	if err != nil {
		return err
	}
	return s.findByKey(ctx, handler, key, record)
}

// 通过条件查询第一个
func (s SQLDao) First(ctx context.Context, where map[string]interface{}, record interface{}) (err error) {
	handler, err := s.GetDbHandler()
	if err != nil {
		return err
	}
	return s.first(ctx, handler, where, record)
}

// 通过条件查询集合
// limit < 0, 查询全部
func (s SQLDao) Find(ctx context.Context, where map[string]interface{}, offset uint, limit uint, records interface{}) (err error) {
	handler, err := s.GetDbHandler()
	if err != nil {
		return err
	}
	return s.find(ctx, handler, where, offset, limit, records)
}

// 新增记录
func (s SQLDao) Insert(ctx context.Context, data map[string]interface{}) (id int64, err error) {
	handler, err := s.GetDbHandler()
	if err != nil {
		return id, err
	}
	i := make([]map[string]interface{}, 0, 1)
	i[0] = data
	return s.insert(ctx, handler, i...)
}

// 通过Key更新
func (s SQLDao) UpdateByKey(ctx context.Context, key interface{}, data map[string]interface{}) (int64, error) {
	handler, err := s.GetDbHandler()
	if err != nil {
		return 0, err
	}
	return s.updateByKey(ctx, handler, key, data)
}

// 通过条件更新
func (s SQLDao) Update(ctx context.Context, where, data map[string]interface{}) (int64, error) {
	handler, err := s.GetDbHandler()
	if err != nil {
		return 0, err
	}
	return s.update(ctx, handler, where, data)
}

// 通过Key删除
func (s SQLDao) DeleteByKey(ctx context.Context, key interface{}) (int64, error) {
	handler, err := s.GetDbHandler()
	if err != nil {
		return 0, err
	}
	return s.deleteByKey(ctx, handler, key)
}

// 删除
func (s SQLDao) Delete(ctx context.Context, where map[string]interface{}) (int64, error) {
	handler, err := s.GetDbHandler()
	if err != nil {
		return 0, err
	}
	return s.delete(ctx, handler, where)
}

// 基于事务新增记录
func (s SQLDao) TXInsert(ctx context.Context, tx *sql.Tx, data map[string]interface{}) (id int64, err error) {
	return s.insert(ctx, tx, data)
}

// 基于事务通过Key更新
func (s SQLDao) TXUpdateByKey(ctx context.Context, tx *sql.Tx, key interface{}, data map[string]interface{}) (int64, error) {
	return s.updateByKey(ctx, tx, key, data)
}

// 基于事务通过条件更新
func (s SQLDao) TXUpdate(ctx context.Context, tx *sql.Tx, where, data map[string]interface{}) (int64, error) {
	return s.update(ctx, tx, where, data)
}

// 基于事务通过Key删除
func (s SQLDao) TXDeleteByKey(ctx context.Context, tx *sql.Tx, key interface{}) (int64, error) {
	return s.deleteByKey(ctx, tx, key)
}

// 基于事务删除
func (s SQLDao) TXDelete(ctx context.Context, tx *sql.Tx, where map[string]interface{}) (int64, error) {
	return s.delete(ctx, tx, where)
}
