package gobase

import (
	"github.com/jmoiron/sqlx"
	"fmt"
)

type TableHelper struct {
	TableName string
	DbKey string
	*sqlx.DB
}

func (th *TableHelper) GetAll(list interface{}, count int) error {
	sql := fmt.Sprintf("select * from %s limit %d", th.TableName, count)

	return th.DB.Select(list, sql)
}

func (th *TableHelper) GetRow() error {
	
	return nil
}