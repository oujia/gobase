package gobase

import (
	"github.com/jmoiron/sqlx"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"reflect"
	"errors"
	"strings"
)

type TableHelper struct {
	TableName string
	DbKey string
	*sqlx.DB
}

type helperResult struct {
	affected int64
	err error
}

const chunkSize = 1000

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

func (th *TableHelper) GetFoundRows() (total int64) {
	sql := "SELECT FOUND_ROWS()"
	row := th.DB.QueryRow(sql)
	row.Scan(&total)

	return
}

func (th *TableHelper) GetCount(where, keyword map[string]interface{}) (int64, error) {
	var total int64
	if keyword == nil {
		keyword = make(map[string]interface{})
	}

	keyword["_field"] = "count(*)"
	delete(keyword, "_sort")
	delete(keyword, "_foundRows")
	err := th.GetOne(&total, where, keyword)
	if err != nil {
		return 0, err
	}

	return total, nil
}

func (th *TableHelper) UpdateObject(data, where map[string]interface{}) (int64, error) {
	if where == nil {
		return 0, errors.New("miss where")
	}

	_where, err := buildWhere(where, "and")
	if err != nil {
		return 0, err
	}
	_data, err := buildWhere(data, ",")
	if err != nil {
		return 0, err
	}
	sql := fmt.Sprintf("update %s set %s where %s", th.TableName, _data, _where)
	fmt.Println(sql)
	rs, err := th.DB.Exec(sql)
	if err != nil {
		return 0, err
	}

	return rs.RowsAffected()
}

func (th *TableHelper) DelObject(where map[string]interface{}) (int64, error) {
	if where == nil {
		return 0, errors.New("miss where")
	}

	_where, err := buildWhere(where, "and")
	if err != nil {
		return 0, err
	}

	sql := fmt.Sprintf("delete from %s where %s", th.TableName, _where)
	fmt.Println(sql)
	rs, err := th.DB.Exec(sql)
	if err != nil {
		return 0, err
	}

	return rs.RowsAffected()
}

func (th *TableHelper) _addObject(obj interface{}, act string) (int64, error) {
	sql := ""
	if act == "replace" {
		sql += "replace into "
	} else if act == "addNx" {
		sql += "insert ignore into "
	} else {
		sql += "insert into "
	}

	sql += th.TableName + " ("
	column, err := getColumnName(obj)
	if err != nil {
		return 0, err
	}
	sql += column + ") values "
	values, err := getColumnValue(obj)
	if err != nil {
		return 0, err
	}

	sql += values
	fmt.Println(sql)

	rs, err := th.DB.Exec(sql)
	if err != nil {
		return 0, err
	}

	return rs.LastInsertId()
}

func (th *TableHelper) AddObject(obj interface{}) (int64, error) {
	return th._addObject(obj, "add");
}

func (th *TableHelper) ReplaceObject(obj interface{}) (int64, error) {
	return th._addObject(obj, "replace");
}

func (th *TableHelper) AddObjectNx(obj interface{}) (int64, error) {
	return th._addObject(obj, "addNx");
}

func (th *TableHelper) _addObjects(objs []interface{}, act string, ch chan helperResult) {
	if len(objs) <= 0 {
		ch <- helperResult{0, errors.New("")}
	}

	sql := ""
	if act == "replace" {
		sql += "replace into "
	} else if act == "addNx" {
		sql += "insert ignore into "
	} else {
		sql += "insert into "
	}

	sql += th.TableName + " ("
	column, err := getColumnName(objs[0])
	if err != nil {
		ch <- helperResult{0, err}
	}
	sql += column + ") values "

	for i := 0; i < len(objs); i++ {
		if i > 0 {
			sql += ", "
		}
		values, err := getColumnValue(objs[i])
		if err != nil {
			ch <- helperResult{0, err}
		}

		sql += values
	}
	fmt.Println(sql)

	rs, err := th.DB.Exec(sql)
	if err != nil {
		ch <- helperResult{0, err}
	}

	af, err := rs.RowsAffected()
	ch <- helperResult{af, err}
}

func (th *TableHelper) _addObjectsWapper(objs interface{}, act string) (int64, error) {
	datas := make([][]interface{}, 0)
	r := reflect.ValueOf(objs)

	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	if r.Kind() != reflect.Slice && r.Kind() != reflect.Array {
		return 0, errors.New("data type must be slice or array")
	}

	for i := 0; i < r.Len(); i += chunkSize {
		end := i + chunkSize
		if end > r.Len() {
			end = r.Len()
		}

		_tmp := make([]interface{}, end-i)
		idx := 0
		for k := i; k < end; k++ {
			_tmp[idx] = r.Index(k).Interface()
			idx++
		}

		datas = append(datas, _tmp)
	}

	var total int64
	errs := make([]string, 0)
	ch := make(chan helperResult)
	defer close(ch)
	for i := 0; i < len(datas); i++ {
		go th._addObjects(datas[i], act, ch) //不保证插入顺序
	}

	for i := 0; i < len(datas); i++ {
		hr := <-ch
		total += hr.affected
		if hr.err != nil {
			errs = append(errs, hr.err.Error())
		}
	}

	var e error
	if len(errs) > 0 {
		e = errors.New(strings.Join(errs, "; "))
	} else {
		e = nil
	}

	return total, e
}

func (th *TableHelper) AddObjects(objs interface{}) (int64, error) {
	return th._addObjectsWapper(objs, "add")
}

func (th *TableHelper) AddObjectsNx(objs interface{}) (int64, error) {
	return th._addObjectsWapper(objs, "addNx")
}

func (th *TableHelper) ReplaceObjects(objs interface{}) (int64, error) {
	return th._addObjectsWapper(objs, "replace")
}

func getColumnName(obj interface{}) (string, error) {
	r := reflect.ValueOf(obj)
	if r.Kind() != reflect.Struct {
		return "", errors.New("add object must be struct")
	}

	t := r.Type()
	column := make([]string, 0)
	for i := 0; i < r.NumField(); i++ {
		dbTag := t.Field(i).Tag.Get("db")
		// 忽略自增零值
		if strings.Contains(dbTag, "ai") && IsZero(r.Field(i)) {
			continue
		}

		column = append(column, strings.Split(dbTag, ",")[0])
	}

	return strings.Join(column, ", "), nil
}

func getColumnValue(obj interface{}) (string, error) {
	r := reflect.ValueOf(obj)
	if r.Kind() != reflect.Struct {
		return "", errors.New("add object must be struct")
	}

	t := r.Type()
	column := make([]string, 0)
	for i := 0; i < r.NumField(); i++ {
		dbTag := t.Field(i).Tag.Get("db")
		// 忽略自增零值
		if strings.Contains(dbTag, "ai") && IsZero(r.Field(i)) {
			continue
		}

		switch r.Field(i).Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16,
			reflect.Int32, reflect.Int64, reflect.Uint,
			reflect.Uint8, reflect.Uint16, reflect.Uint32,
			reflect.Uint64, reflect.Uintptr:
			column = append(column, fmt.Sprintf("%d", r.Field(i)))
		case reflect.String:
			column = append(column, fmt.Sprintf("'%s'", r.Field(i)))
		case reflect.Float32, reflect.Float64:
			column = append(column, fmt.Sprintf("%f", r.Field(i)))

		default:
			return "", errors.New(fmt.Sprintf("sql params[column=%s] error", dbTag))
		}
	}

	return "(" + strings.Join(column, ", ") + ")", nil
}

func (th *TableHelper) buildSql(where, keyword map[string]interface{}) (string, error) {
	field := "*"
	if _field, ok := keyword["_field"]; ok {
		field = _field.(string)
	}

	if _fr, ok := keyword["_foundRows"]; ok && _fr.(bool) {
		field = "SQL_CALC_FOUND_ROWS " + field
	}

	sql := fmt.Sprintf("select %s from %s where ", field, th.TableName)
	if where != nil {
		_where, err := buildWhere(where, "and")
		if err != nil {
			return "", err
		}
		sql += _where
	} else {
		sql += " 1"
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

func buildWhere(where map[string]interface{}, sep string) (string, error) {
	sqlSlice := make([]string, 0)

	for k, v := range where {
		r := reflect.ValueOf(v)

		switch r.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16,
			reflect.Int32, reflect.Int64, reflect.Uint,
			reflect.Uint8, reflect.Uint16, reflect.Uint32,
			reflect.Uint64, reflect.Uintptr:
			sqlSlice = append(sqlSlice, fmt.Sprintf("%s=%d",k, v))
		case reflect.String:
			sqlSlice = append(sqlSlice, fmt.Sprintf("%s='%s'", k, v))
		case reflect.Float32, reflect.Float64:
			sqlSlice = append(sqlSlice, fmt.Sprintf("%s=%f", k, v))
		case reflect.Array, reflect.Slice:
			inSql := fmt.Sprintf("%s in (", k)

			for i := 0; i < r.Len() ; i++ {
				if i != 0 {
					inSql += ", "
				}
				switch r.Index(i).Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16,
					reflect.Int32, reflect.Int64, reflect.Uint,
					reflect.Uint8, reflect.Uint16, reflect.Uint32,
					reflect.Uint64, reflect.Uintptr:
					inSql += fmt.Sprintf("%d", r.Index(i))
				case reflect.String:
					inSql += fmt.Sprintf("'%s'", r.Index(i))
				case reflect.Float32, reflect.Float64:
					inSql += fmt.Sprintf("%f", r.Index(i))
				default:
					return "", errors.New(fmt.Sprintf("sql params[key=%s] error", k))
				}
			}

			inSql += ")"
			sqlSlice = append(sqlSlice, inSql)

		default:
			return "", errors.New(fmt.Sprintf("sql params[key=%s] error", k))
		}
	}

	return strings.Join(sqlSlice, fmt.Sprintf(" %s ", sep)), nil
}

