package gobase

import (
	"github.com/go-redis/redis"
)

type R2M struct {
	*TableHelper
	Redis *redis.Client
}

func (r *R2M) GetAll(list interface{}, where, keyword map[string]interface{}) error {
	// TODO cache
	return r.TableHelper.GetAll(list, where, keyword)
}

func (r *R2M) GetRow(item interface{}, where, keyword map[string]interface{}) error {
	// TODO cache
	return r.TableHelper.GetRow(item, where, keyword)
}