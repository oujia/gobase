package gobase

import (
	"github.com/jmoiron/sqlx"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"reflect"
	"errors"
)

type TableHelper struct {
	TableName string
	DbKey string
	*sqlx.DB
}

// 读取数据
// keyword 查询关键字, ['_field', '_where', '_limit', '_sort', '_groupby']
// list type pointer to data struct slice
func (th *TableHelper) GetAll(list interface{}, where, keyword map[string]interface{}) error {
	sql, err := th.buildSql(where, keyword)
	if err != nil {
		return err
	}

	fmt.Println(sql)
	return th.DB.Select(list, sql)
}

func (th *TableHelper) GetRow(item interface{}, where, keyword map[string]interface{}) error {
	if keyword != nil {
		keyword["_limit"] = 1
	}

	sql, err := th.buildSql(where, keyword)
	if err != nil {
		return err
	}

	fmt.Println(sql)
	return th.DB.Get(item, sql)
}

// 返回SQL语句执行结果集中的第一行第一列数据
// result type base
func (th *TableHelper) GetOne(result interface{}, where, keyword map[string]interface{}) error {
	return th.GetRow(result, where, keyword)
}

// 返回SQL语句执行结果集中的第一列数据
// result type pointer to slice
func (th *TableHelper) GetCol(result interface{}, where, keyword map[string]interface{}) error {
	return th.GetAll(result, where, keyword)
}

func (th *TableHelper) GetFoundRows() (total int) {
	sql := "SELECT FOUND_ROWS()"
	row := th.DB.QueryRow(sql)
	row.Scan(&total)

	return
}

func (th *TableHelper) buildSql(where, keyword map[string]interface{}) (string, error) {
	field := "*"
	if _field, ok := keyword["_field"]; ok {
		field = _field.(string)
	}

	if _fr, ok := keyword["_foundRows"]; ok && _fr.(bool) {
		field = "SQL_CALC_FOUND_ROWS " + field
	}

	sql := fmt.Sprintf("select %s from %s where 1=1", field, th.TableName)
	for k, v := range where {
		r := reflect.ValueOf(v)

		switch r.Kind() {
		case reflect.Int:
			sql += fmt.Sprintf(" and %s=%d", k, v)
		case reflect.String:
			sql += fmt.Sprintf(" and %s='%s'", k, v)
		case reflect.Float32:
			fallthrough
		case reflect.Float64:
			sql += fmt.Sprintf(" and %s=%f", k, v)
		case reflect.Array:
			fallthrough
		case reflect.Slice:
			sql += fmt.Sprintf(" and %s in (", k)

			for i := 0; i < r.Len() ; i++ {
				if i != 0 {
					sql += ", "
				}
				if r.Index(i).Kind() == reflect.String {
					sql += fmt.Sprintf("'%s'", r.Index(i))
				} else if r.Index(i).Kind() == reflect.Int {
					sql += fmt.Sprintf("%d", r.Index(i))
				} else {
					return "", errors.New(fmt.Sprintf("sql params[key=%s] error", k))
				}
			}

			sql += ")"
		}
	}

	if _where, ok := keyword["_where"]; ok {
		sql += " and " + _where.(string)
	}

	if _sort, ok := keyword["_sort"]; ok {
		sql += fmt.Sprintf(" order by %s", _sort.(string))
	}

	if _group, ok := keyword["_group"]; ok {
		sql += fmt.Sprintf(" group by %s", _group.(string))
	}

	if _limit, ok := keyword["_limit"]; ok {
		_format := " limit %s"
		if reflect.ValueOf(_limit).Kind() == reflect.Int {
			_format = " limit %d"
		}
		sql += fmt.Sprintf(_format, _limit)
	}

	return sql, nil
}

