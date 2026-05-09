package clickhouse

import (
	"fmt"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/spf13/viper"
)

var (
	CK *clickhImpl
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
	maxIdleConns := viper.GetInt("maxIdleConns")
	maxOpenConns := viper.GetInt("maxOpenConns")
	connMaxLifetime := viper.GetDuration("connMaxLifetime")

	db := clickhouse.OpenDB(&clickhouse.Options{
		Addr: []string{address},
		Auth: clickhouse.Auth{
			Database: database,
			Username: username,
			Password: password,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": viper.GetInt("max_execution_time"),
		},
		DialTimeout: time.Duration(viper.GetInt("connection_timeout")) * time.Second,
		ReadTimeout: time.Duration(viper.GetInt("read_timeout")) * time.Second,
		Debug:       viper.GetBool("debug"),
	})
	if maxIdleConns > 0 {
		db.SetMaxIdleConns(maxIdleConns)
	}
	if maxOpenConns > 0 {
		db.SetMaxOpenConns(maxOpenConns)
	}
	if connMaxLifetime > 0 {
		db.SetConnMaxLifetime(connMaxLifetime)
	}
	if err := db.Ping(); err != nil {
		panic(err)
	}

	CK = &clickhImpl{DB: db}
}
