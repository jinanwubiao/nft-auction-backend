package config

import (
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

type Config struct {
	Server     *Server     `toml:"server" json:"server"`
	ProjectCfg *ProjectCfg `toml:"project_cfg" mapstructure:"project_cfg" json:"project_cfg"`
	Log        LogConf     `toml:"log" json:"log"`
	DB         *DBConfig   `toml:"db" json:"db"`
	EthCfg     *EthConf
}

type Server struct {
	Port   string `toml:"port" json:"port"`
	MaxNum int64  `toml:"max_num" json:"max_num"`
}

type ProjectCfg struct {
	Name string `toml:"name" mapstructure:"name" json:"name"`
}

type DBConfig struct {
	User               string `toml:"user" json:"user"`                                                                        // 用户
	Password           string `toml:"password" json:"password"`                                                                // 密码
	Host               string `toml:"host" json:"host"`                                                                        // 地址
	Port               int    `toml:"port" json:"port"`                                                                        // 端口
	Database           string `toml:"database" json:"database"`                                                                // 数据库
	MaxIdleConns       int    `toml:"max_idle_conns" mapstructure:"max_idle_conns" json:"max_idle_conns"`                      // 最大空闲连接数
	MaxOpenConns       int    `toml:"max_open_conns" mapstructure:"max_open_conns" json:"max_open_conns"`                      // 最大打开连接数
	MaxConnMaxLifetime int64  `toml:"max_conn_max_lifetime" mapstructure:"max_conn_max_lifetime" json:"max_conn_max_lifetime"` // 连接复用时间
	LogLevel           string `toml:"log_level" mapstructure:"log_level" json:"log_level"`                                     // 日志级别，枚举（info、warn、error和silent）
}

type LogConf struct {
	ServiceName string `toml:"service_name" mapstructure:"service_name" json:"service_name"`
	Mode        string `toml:"mode" json:"mode"`
	Path        string `toml:"path" json:"path"`
	Level       string `toml:"level" json:"level"`
	Compress    bool   `toml:"compress" json:"compress"`
	KeepDays    int    `toml:"keep_days" mapstructure:"keep_days" json:"keep_days"`
}

type EthConf struct {
	RpcUrl  string `toml:"rpc_url" mapstructure:"rpc_url" json:"rpc_url"`
	ChainId uint64 `toml:"chain_id" mapstructure:"chain_id" json:"chain_id"`
}

func Load(configFilePath string) (*Config, error) {
	godotenv.Load()
	viper.SetConfigFile(configFilePath)
	viper.SetConfigType("toml")
	viper.AutomaticEnv()
	viper.SetEnvPrefix("NFT")
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}
	config, err := DefaultConfig()
	if err != nil {
		return nil, err
	}

	if err := viper.Unmarshal(config); err != nil {
		return nil, err
	}
	return config, nil
}

func DefaultConfig() (*Config, error) {
	return &Config{}, nil
}
