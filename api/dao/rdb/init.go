package rdb

import (
	"context"
	"fmt"

	redis "github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
)

var (
	RdbRedisClient      redis.UniversalClient
	RdbBrc20StateClient redis.UniversalClient
)

func InitClients() {
	RdbRedisClient = Init("conf/kvdb.yaml")
	RdbBrc20StateClient = Init("conf/api/kvdb_brc20.yaml")
}

func InitClient(filename string) (rds *redis.Client) {
	viper.SetConfigFile(filename)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			panic(fmt.Errorf("fatal error config file: %s", err))
		} else {
			panic(fmt.Errorf("fatal error config file: %s", err))
		}
	}

	addr := viper.GetString("addr")
	password := viper.GetString("password")
	database := viper.GetInt("database")
	dialTimeout := viper.GetDuration("dialTimeout")
	readTimeout := viper.GetDuration("readTimeout")
	writeTimeout := viper.GetDuration("writeTimeout")
	idleTimeout := viper.GetDuration("idleTimeout")
	idleCheckFrequency := viper.GetDuration("idleCheckFrequency")
	poolSize := viper.GetInt("poolSize")
	rds = redis.NewClient(&redis.Options{
		Addr:               addr,
		Password:           password,
		DB:                 database,
		DialTimeout:        dialTimeout,
		ReadTimeout:        readTimeout,
		WriteTimeout:       writeTimeout,
		PoolSize:           poolSize,
		IdleTimeout:        idleTimeout,
		IdleCheckFrequency: idleCheckFrequency,
	})
	if err := rds.Ping(context.Background()).Err(); err != nil {
		panic(err)
	}
	return rds
}

func Init(filename string) (rds redis.UniversalClient) {
	viper.SetConfigFile(filename)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			panic(fmt.Errorf("fatal error config file: %s", err))
		} else {
			panic(fmt.Errorf("fatal error config file: %s", err))
		}
	}

	addrs := viper.GetStringSlice("addrs")
	password := viper.GetString("password")
	database := viper.GetInt("database")
	dialTimeout := viper.GetDuration("dialTimeout")
	readTimeout := viper.GetDuration("readTimeout")
	writeTimeout := viper.GetDuration("writeTimeout")
	idleTimeout := viper.GetDuration("idleTimeout")
	idleCheckFrequency := viper.GetDuration("idleCheckFrequency")
	poolSize := viper.GetInt("poolSize")
	rds = redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:              addrs,
		Password:           password,
		DB:                 database,
		DialTimeout:        dialTimeout,
		ReadTimeout:        readTimeout,
		WriteTimeout:       writeTimeout,
		PoolSize:           poolSize,
		IdleTimeout:        idleTimeout,
		IdleCheckFrequency: idleCheckFrequency,
	})
	if err := rds.Ping(context.Background()).Err(); err != nil {
		panic(err)
	}
	return rds
}
