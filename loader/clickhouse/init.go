package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/spf13/viper"
)

var (
	CK driver.Conn
)

func Init() {
	viper.SetConfigFile("conf/db.yaml")
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		} else {
			panic(fmt.Errorf("Fatal error config file: %s \n", err))
		}
	}

	address := viper.GetString("address")
	username := viper.GetString("username")
	password := viper.GetString("password")
	database := viper.GetString("database")

	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{address},
		Auth: clickhouse.Auth{
			Database: database,
			Username: username,
			Password: password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": viper.GetInt("max_execution_time"),
		},
		DialTimeout:      time.Duration(viper.GetInt("connection_timeout")) * time.Second,
		ReadTimeout:      time.Duration(viper.GetInt("read_timeout")) * time.Second,
		MaxOpenConns:     viper.GetInt("maxOpenConns"),
		MaxIdleConns:     viper.GetInt("maxIdleConns"),
		ConnMaxLifetime:  viper.GetDuration("connMaxLifetime"),
		ConnOpenStrategy: clickhouse.ConnOpenInOrder,
		BlockBufferSize:  10,
	})

	if err != nil {
		panic(err)
	}

	if err := conn.Ping(context.Background()); err != nil {
		panic(err)
	}

	CK = conn

	// Initialize HTTP client for RowBinary bulk inserts
	InitHTTP(address, database, username, password, time.Duration(viper.GetInt("write_timeout"))*time.Second)
}
