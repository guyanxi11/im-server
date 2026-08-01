// Package config 负责加载与解析 im-server 的运行配置
// 作者: wym
// 从项目根目录的 config.yaml 读取配置（server/mysql/jwt 三段）
package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// ServerConfig 是 HTTP 服务相关配置
type ServerConfig struct {
	Addr string `mapstructure:"addr"`
}

// MySQLConfig 是数据库连接配置
type MySQLConfig struct {
	// DSN 是 GORM 使用的数据源连接串，格式见 config.example.yaml
	DSN string `mapstructure:"dsn"`
}

// JWTConfig 是 JWT 签发相关配置
type JWTConfig struct {
	// Secret 是签名密钥，生产环境必须替换为随机长字符串
	Secret string `mapstructure:"secret"`
	// ExpireHours 是 token 有效期（小时）
	ExpireHours int `mapstructure:"expire_hours"`
}

// RedisConfig 是 Redis 连接配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	// DB 用独立编号隔离本项目的 key，避免与共用同一 Redis 实例的其他项目冲突
	DB int `mapstructure:"db"`
}

// Config 是 im-server 的完整运行配置
type Config struct {
	Server ServerConfig `mapstructure:"server"`
	MySQL  MySQLConfig  `mapstructure:"mysql"`
	JWT    JWTConfig    `mapstructure:"jwt"`
	Redis  RedisConfig  `mapstructure:"redis"`
}

// Load 从项目根目录的 config.yaml 加载配置
// 找不到文件或解析失败会直接 panic，因为服务缺少配置无法正常运行
func Load() *Config {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Errorf("读取 config.yaml 失败，请复制 config.example.yaml 为 config.yaml 并按本地环境修改: %w", err))
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("解析 config.yaml 失败: %w", err))
	}

	return &cfg
}
