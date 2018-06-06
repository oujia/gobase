package gobase

import (
	"fmt"
	"strings"
	"reflect"
	"crypto/md5"
	"github.com/gomodule/redigo/redis"
)

type R2M struct {
	*TableHelper
	Redis *redis.Pool
	R2mInfo
}

func (r *R2M)getPreKey(where map[string]interface{}, prefix string, trimWhere bool) string {
	var key string
	if "row" == prefix {
		key = r.R2mInfo.Key
	} else {
		key = r.R2mInfo.AllKey
	}
	cacheKey := r.basePre(prefix)
	cacheKeys := make([]string, 0, len(where))
	if key != "" {
		keys := strings.Split(key, ",")
		for _, v := range keys {
			v = strings.TrimSpace(v)
			if wv, ok := where[v]; ok {
				cacheKeys = append(cacheKeys, getKeyItem(v, wv))
			} else if prefix == "row" {
				return ""
			} else {
				return cacheKey + ":others"
			}
		}

		if trimWhere {
			for _, v := range keys {
				delete(where, v)
			}
		}
	}

	cacheKeys = cacheKeys[:]
	if len(cacheKeys) > 0 {
		return cacheKey + ":" + strings.Join(cacheKeys, ":")
	} else {
		return cacheKey + ":others"
	}
}

func (r *R2M) getMD5Key(where map[string]interface{}, prefix string, keyword map[string]interface{}) string {
	cacheKey := r.getPreKey(where, prefix, true)

	var f = func(k, v string) string {
		return fmt.Sprintf("%s[]=%s", k, v)
	}

	key, _ := JoinMapInterface(where, "%s=%s", "&", f, "&")
	if len(keyword) > 0 {
		strKeyword, err := JoinMapInterface(keyword, "%s=%s", "&", f, "&")
		if err == nil {
			key += "&" + strKeyword
		}
	}

	if len(key) > 32 {
		key = fmt.Sprintf("%x", md5.Sum([]byte(key)))
	}

	if len(key) > 0 {
		cacheKey += ":" + key
	} else {
		cacheKey += ":empty"
	}

	return cacheKey
}

func getKeyItem(key string, value interface{}) string {
	r := reflect.ValueOf(value)
	var rs string
	switch r.Kind() {
	case reflect.Slice, reflect.Array:
		var tmp string
		for i := 0; i < r.Len() ; i++ {
			if i > 0 {
				tmp += ","
			}
			switch r.Index(i).Kind() {
			case reflect.Int, reflect.Int8, reflect.Int16,
				reflect.Int32, reflect.Int64, reflect.Uint,
				reflect.Uint8, reflect.Uint16, reflect.Uint32,
				reflect.Uint64, reflect.Uintptr:
				tmp += fmt.Sprintf("%d", r.Index(i))
			case reflect.String:
				tmp += fmt.Sprintf("%s", r.Index(i))
			case reflect.Float32, reflect.Float64:
				tmp += fmt.Sprintf("%f", r.Index(i))
			}
		}
		if len(tmp) > 32 {
			tmp = fmt.Sprintf("%x", md5.Sum([]byte(tmp)))
		}
		rs = fmt.Sprintf("%s=%s", key, tmp)
	case reflect.String:
		rs = fmt.Sprintf("%s=%s", key, value)
	case reflect.Int, reflect.Int8, reflect.Int16,
		reflect.Int32, reflect.Int64, reflect.Uint,
		reflect.Uint8, reflect.Uint16, reflect.Uint32,
		reflect.Uint64, reflect.Uintptr:
		rs = fmt.Sprintf("%s=%d", key, value)
	case reflect.Float32, reflect.Float64:
		rs = fmt.Sprintf("%s=%f", key, value)
	}

	return rs
}

func (r *R2M)basePre(prefix string) string {
	return fmt.Sprintf("%s:%s", r.DbKey, prefix)
}

func (r *R2M) GetAll(list interface{}, where, keyword map[string]interface{}) error {
	// TODO cache
	return r.TableHelper.GetAll(list, where, keyword)
}

func (r *R2M) GetRow(item interface{}, where, keyword map[string]interface{}) error {
	cacheKey := r.getPreKey(where, "row", false)
	var aliasCacheKey string
	if cacheKey == "" {
		aliasCacheKey = r.getMD5Key(where, "alias", nil)
		cacheKey, _ = redis.String(r.Redis.Get().Do("GET", aliasCacheKey))
	}

	reply, err := redis.Values(r.Redis.Get().Do("HGETALL", cacheKey))
	if err != nil {
		return err
	}

	if len(reply) > 0 { //hint
		err = redis.ScanStruct(reply, item)
		if err != nil {
			return err
		}
	} else {
		err = r.TableHelper.GetRow(item, where, keyword)
		if err != nil {
			return err
		}

		err = r.setStructCache(item, cacheKey)
		if err != nil {
			return err
		}
	}

	if aliasCacheKey != "" {

	}

	return nil
}

func (r *R2M) setStructCache(data interface{}, cacheKey string) error {
	conn := r.Redis.Get()
	_, err := conn.Do("HMSET", redis.Args{}.Add(cacheKey).AddFlat(data)...)
	if err != nil {
		return err
	}

	if r.R2mInfo.TTL > 0 {
		_, err := conn.Do("EXPIRE", cacheKey, r.R2mInfo.TTL)
		if err != nil {
			return err
		}
	}

	return nil
}