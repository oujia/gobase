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

func (th *TableHelper) GetAll(list interface{}, where, keyword map[string]interface{}) error {
	sql, err := th.buildSql(where, keyword)
	if err != nil {
		return err
	}

	fmt.Println(sql)
	return th.DB.Select(list, sql)
}

func (th *TableHelper) GetRow(item interface{}, where, keyword map[string]interface{}) error {
	keyword["_limit"] = 1

	sql, err := th.buildSql(where, keyword)
	if err != nil {
		return err
	}

	fmt.Println(sql)
	return th.DB.Get(item, sql)
}

func (th *TableHelper) GetOne(result interface{}, where, keyword map[string]interface{}) error {
	keyword["_limit"] = 1

	sql, err := th.buildSql(where, keyword)
	if err != nil {
		return err
	}

	fmt.Println(sql)
	row := th.DB.QueryRow(sql)
	err = row.Scan(result)
	if err != nil {
		return err
	}

	return nil
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