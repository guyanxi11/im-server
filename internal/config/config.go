// Package config 负责加载与解�?im-server 的运行配�?// 作�? wym
// 从项目根目录�?config.yaml 读取配置（server/mysql/jwt 三段�?package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// ServerConfig �?HTTP 服务相关配置
type ServerConfig struct {
	Addr string `mapstructure:"addr"`
}

// MySQLConfig 是数据库连接配置
type MySQLConfig struct {
	// DSN �?GORM 使用的数据源连接串，格式�?config.example.yaml
	DSN string `mapstructure:"dsn"`
}

// JWTConfig �?JWT 签发相关配置
type JWTConfig struct {
	// Secret 是签名密钥，生产环境必须替换为随机长字符�?	Secret string `mapstructure:"secret"`
	// ExpireHours �?token 有效期（小时�?	ExpireHours int `mapstructure:"expire_hours"`
}

// RedisConfig �?Redis 连接配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	// DB 用独立编号隔离本项目�?key，避免与共用同一 Redis 实例的其他项目冲�?	DB int `mapstructure:"db"`
}

// Config �?im-server 的完整运行配�?type Config struct {
	Server ServerConfig `mapstructure:"server"`
	MySQL  MySQLConfig  `mapstructure:"mysql"`
	JWT    JWTConfig    `mapstructure:"jwt"`
	Redis  RedisConfig  `mapstructure:"redis"`
}

// Load 从项目根目录�?config.yaml 加载配置
// 找不到文件或解析失败会直�?panic，因为服务缺少配置无法正常运�?func Load() *Config {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		panic(fmt.Errorf("读取 config.yaml 失败，请复制 config.example.yaml �?config.yaml 并按本地环境修改: %w", err))
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		panic(fmt.Errorf("解析 config.yaml 失败: %w", err))
	}

	return &cfg
}
