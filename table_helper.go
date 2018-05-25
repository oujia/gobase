package gobase

import (
	"github.com/jmoiron/sqlx"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

type TableHelper struct {
	TableName string
	DbKey string
	*sqlx.DB
}

func (th *TableHelper) GetAll(list interface{}, field string, count int) error {
	if field == "" {
		field = "*"
	}
	sql := fmt.Sprintf("select %s from %s limit %d", field, th.TableName, count)
	fmt.Println(sql)
	return th.DB.Select(list, sql)
}

func (th *TableHelper) GetRow() error {
	
	return nil
}