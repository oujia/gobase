package gobase

import (
"os"
"fmt"
"encoding/json"
)

type DbInfo struct {
	Host string `json:"dbHost"`
	Port int `json:"dbPort,string"`
	Name string `json:"dbName"`
	User string `json:"dbUser"`
	Pass string `json:"dbPass"`
}

type RedisInfo struct {
	Host string `json:"host"`
	Port int `json:"port,string,omitempty"`
	Pwd string `json:"pwd"`
	Db int `json:"db,string,omitempty"`
	Timeout int `json:"connect_timeout,string,omitempty"`
}

type R2mInfo struct {
	AllKey string `json:"all_key,omitempty"`
	Key string `json:"key"`
	Ttl int `json:"ttl,string,omitempty"`
}

type GlobalConf struct {
	DbInfo map[string]DbInfo `json:"dbInfo"`
	RedisInfo map[string]RedisInfo `json:"redisInfo"`
	R2mInfo map[string]R2mInfo `json:"r2mInfo"`
}

func LoadGlobalConf(path string) (*GlobalConf, error)  {
	configFile, err := os.Open(path)

	if err != nil {
		return nil, fmt.Errorf("Unable to read configuration file %s", path)
	}

	conf := new(GlobalConf)
	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&conf)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse configuration file %s\n%s", path, err.Error())
	}

	return conf, nil
}


