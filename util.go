package gobase

import (
	"net"
	"errors"
	"time"
	"github.com/gomodule/redigo/redis"
	"fmt"
	"github.com/jmoiron/sqlx"
	"reflect"
	"strings"
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

func NewRedisPool(redisInfo *RedisInfo) *redis.Pool {
	server := fmt.Sprintf("%s:%d", redisInfo.Host, redisInfo.Port)

	return &redis.Pool{

		MaxIdle:     5,
		IdleTimeout: 240 * time.Second,

		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if len(redisInfo.Pwd) > 0 {
				_, err := c.Do("AUTH", redisInfo.Pwd);
				if err != nil {
					c.Close()
					return nil, err
				}
			}

			if _, err := c.Do("SELECT", redisInfo.Db); err != nil {
				c.Close()
				return nil, err
			}

			return c, err
		},

		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
	}
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

type inFormatFunc func(k, v string) string

func JoinMapInterface(data map[string]interface{}, format string, sep string, f inFormatFunc, inSep string) (string, error) {
	dataSlice := make([]string, 0, len(data))

	for k, v := range data {
		r := reflect.ValueOf(v)

		switch r.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16,
			reflect.Int32, reflect.Int64, reflect.Uint,
			reflect.Uint8, reflect.Uint16, reflect.Uint32,
			reflect.Uint64, reflect.Uintptr:
			dataSlice = append(dataSlice, fmt.Sprintf(format, k, fmt.Sprintf("%d", v)))
		case reflect.String:
			dataSlice = append(dataSlice, fmt.Sprintf(format, k, v))
		case reflect.Float32, reflect.Float64:
			dataSlice = append(dataSlice, fmt.Sprintf(format, k, fmt.Sprintf("%f", v)))
		case reflect.Array, reflect.Slice:
			inSlice := make([]string, 0, r.Len())
			for i := 0; i < r.Len(); i++ {
				switch r.Index(i).Kind() {
				case reflect.Int, reflect.Int8, reflect.Int16,
					reflect.Int32, reflect.Int64, reflect.Uint,
					reflect.Uint8, reflect.Uint16, reflect.Uint32,
					reflect.Uint64, reflect.Uintptr:
					inSlice = append(inSlice, f(k, fmt.Sprintf("%d", r.Index(i))))
				case reflect.String:
					inSlice = append(inSlice, f(k, fmt.Sprintf("%s", r.Index(i))))
				case reflect.Float32, reflect.Float64:
					inSlice = append(inSlice, f(k, fmt.Sprintf("%f", r.Index(i))))
				default:
					return "", errors.New(fmt.Sprintf("params[key=%s] error", k))
				}
			}
			if len(StringFilter(inSlice, notEmptyString)) > 0 {
				dataSlice = append(dataSlice, strings.Join(inSlice[:], inSep))
			}

		default:
			return "", errors.New(fmt.Sprintf("params[key=%s] error", k))
		}

	}
	return strings.Join(dataSlice[:], sep), nil
}

func emptyString(str string) bool {
	return len(str) == 0
}

func notEmptyString(str string) bool {
	return len(str) > 0
}

func StringFilter(vs []string, f func(string) bool) []string {
	vsf := make([]string, 0, len(vs))
	for _, v := range vs {
		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf[:]
}