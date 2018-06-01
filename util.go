package gobase

import (
	"net"
	"errors"
	"time"
	"github.com/go-redis/redis"
	"fmt"
	"github.com/jmoiron/sqlx"
	"reflect"
)

func ExternalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "unknown", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "unknown", errors.New("are you connected to the network?")
}

func FormatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func NewRedisClient(redisInfo *RedisInfo) *redis.Client {
	addr := fmt.Sprintf("%s:%d", redisInfo.Host, redisInfo.Port)

	return redis.NewClient(&redis.Options{
		Addr: addr,
		Password: redisInfo.Pwd,
		DB: redisInfo.Db,
	})
}

func NewDbClient(dbName string, info map[string]DbInfo) (*sqlx.DB, error) {
	dbInfo, ok := info[dbName];
	if !ok {
		return nil, errors.New(fmt.Sprintf("lost [%s] dbinfo", dbName))
	}

	var dataSource string
	if len(dbInfo.Pass) > 0 {
		dataSource = fmt.Sprintf("%s:%s@(%s:%d)/%s?charset=utf8&parseTime=true", dbInfo.User, dbInfo.Pass, dbInfo.Host, dbInfo.Port, dbInfo.Name)
	} else {
		dataSource = fmt.Sprintf("%s@(%s:%d)/%s?charset=utf8&parseTime=true", dbInfo.User, dbInfo.Host, dbInfo.Port, dbInfo.Name)
	}

	//fmt.Println(dataSource)
	return sqlx.Connect("mysql", dataSource)
}

func IsZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	}
	return false
}

func ToSnake(s string) string {
	b := make([]byte, 0, len(s)*2)
	j := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			if i > 0 && j{
				b = append(b, '_')
			}
			c += 'a' - 'A'
		}
		j = c != '_'
		b = append(b, c)
	}

	return string(b[:])
}