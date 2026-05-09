package rdb

import (
	"context"
	"fmt"
	"fractal-indexer/logger"

	redis "github.com/go-redis/redis/v8"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var (
	useCluster bool
	RdbClient  redis.UniversalClient
	ctx        = context.Background()
)

func InitClient(filename string) (rds *redis.Client) {
	viper.SetConfigFile(filename)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		} else {
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
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
	return rds
}

func Init(filename string) (rds redis.UniversalClient) {
	viper.SetConfigFile(filename)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		} else {
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
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
	useCluster := viper.GetBool("useCluster")
	if useCluster {
		rds = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs:              addrs,
			Password:           password,
			DialTimeout:        dialTimeout,
			ReadTimeout:        readTimeout,
			WriteTimeout:       writeTimeout,
			PoolSize:           poolSize,
			IdleTimeout:        idleTimeout,
			IdleCheckFrequency: idleCheckFrequency,
		})
	} else {
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
	}

	return rds
}

func FlushdbInRedis() {
	logger.Log.Info("FlushdbInRedis start")

	flushDb := func(cli redis.UniversalClient, name string) {
		if cc, ok := cli.(*redis.ClusterClient); ok {
			logger.Log.Info("FlushDbInRedis ForEachMaster", zap.String("name", name))
			cc.ForEachMaster(ctx, func(ctx context.Context, master *redis.Client) error {
				if err := master.FlushDB(ctx).Err(); err != nil {
					logger.Log.Panic("FlushDbInRedis ForEachMaster", zap.String("name", name), zap.Error(err))
				}
				return nil
			})
		} else {
			logger.Log.Info("FlushDbInRedis", zap.String("name", name))
			if err := cli.FlushDB(ctx).Err(); err != nil {
				logger.Log.Panic("FlushDbInRedis", zap.String("name", name), zap.Error(err))
			}
		}
	}

	// todo: pika cluster flushdb
	flushDb(RdbClient, "utxo")
	logger.Log.Info("FlushdbInRedis finish")
}
