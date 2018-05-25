package gobase

import (
	"net"
	"errors"
	"time"
	"github.com/go-redis/redis"
	"fmt"
	"github.com/jmoiron/sqlx"
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

	fmt.Println(dataSource)
	return sqlx.Connect("mysql", dataSource)
}