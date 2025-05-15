package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server ServerConfig
	Log    LogConfig
}

/*
[server]
host = "127.0.0.1"
port = 8080
dir = "public"
token = ""
*/
type ServerConfig struct {
	Port  int    `toml:"port"`
	Host  string `toml:"host"`
	Dir   string `toml:"dir"`
	Token string `toml:"token"`
}

/*
[log]
logFilePath = "nemu.log"
maxLogSize = 5
level = "info"
*/
type LogConfig struct {
	LogFilePath string `toml:"logFilePath"`
	MaxLogSize  int    `toml:"maxLogSize"`
	Level       string `toml:"level"`
}

// LoadConfig 从 TOML 配置文件加载配置
func LoadConfig(filePath string) (*Config, error) {
	if !FileExists(filePath) {
		// 楔入配置文件
		err := DefaultConfig().WriteConfig(filePath)
		if err != nil {
			return nil, err
		}
		return DefaultConfig(), nil
	}

	var config Config
	if _, err := toml.DecodeFile(filePath, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// 写入配置文件
func (c *Config) WriteConfig(filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := toml.NewEncoder(file)
	return encoder.Encode(c)
}

// 检测文件是否存在
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

// 默认配置结构体
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:  8168,
			Host:  "127.0.0.1",
			Dir:   "public",
			Token: "",
		},
		Log: LogConfig{
			LogFilePath: "nemu.log",
			MaxLogSize:  5,
			Level:       "info",
		},
	}
}
