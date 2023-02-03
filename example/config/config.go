package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/qumogu/go-tools/mongodb"
)

var Conf Config

// 配置
type Config struct {
	ServerRunMode   string            `json:"server_run_mode" yaml:"server_run_mode"`   // gin的启动模式
	Profile         bool              `json:"profile" yaml:"profile"`                   // 是否开启 profile
	HttpPort        string            `json:"http_port" yaml:"http_port"`               // "8080"
	GracefulTimeout int               `json:"graceful_timeout" yaml:"graceful_timeout"` // 停止超时时间
	Mongo           mongodb.MongoConf `json:"mongo_conf" yaml:"mongo_conf"`
}

// Parse 解析环境变量/配置文件 获取配置.
func Parse() error {
	Conf.ServerRunMode = getEnvValue("SERVER_RUN_MODE", "debug")
	Conf.Profile = getEnvBool("PROFILE", false)
	Conf.HttpPort = getEnvValue("HTTP_PORT", "6677")
	Conf.GracefulTimeout = getEnvInt("GRACEFUL_TIMEOUT", 10)
	Conf.Mongo.Addr = getEnvValue("MONGODB_ADDR", "172.28.100.35:27017")
	Conf.Mongo.Database = getEnvValue("MONGODB_DATABASE", "bruce-test")
	Conf.Mongo.UserName = getEnvValue("MONGODB_USER_NAME", "admin")
	Conf.Mongo.PWD = getEnvValue("MONGODB_PWD", "JXdbpd37Q2Ayo") // JXdbpd37Q2Ayo
	Conf.Mongo.URI = fmt.Sprintf("mongodb://%s:%s@%s", Conf.Mongo.UserName, Conf.Mongo.PWD, Conf.Mongo.Addr)
	mongodb.InitMongoManager(Conf.Mongo)

	return nil
}

func getEnvValue(key, def string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}

	return def
}

func getEnvInt(key string, def int) int {
	if value, ok := os.LookupEnv(key); ok {
		n, err := strconv.Atoi(value)
		if err != nil {
			fmt.Println("get var from env parse to int failed, key:", key, "error:", err)
			return def
		}

		return n
	}

	return def
}

func getEnvBool(key string, def bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		return value == "true"
	}

	return def
}
