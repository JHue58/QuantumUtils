package logger

import "go.uber.org/zap/zapcore"

// QLogConfig QLogger配置文件
type QLogConfig struct {
	// 日志等级
	Level zapcore.Level
	// 每个日志文件保存的大小 M
	MaxSize int
	// 最长保存多少天
	MaxAge int
	// 最大备份数量
	MaxBackups int
	// 是否压缩
	Compress bool
}

var logConfMap = make(map[string]*QLogConfig)

// SetLogConfig 设置日志配置
func SetLogConfig(name string, config QLogConfig) {
	logConfMap[name] = &config
}

// GetLogConfig 获取日志配置
func GetLogConfig(name string) *QLogConfig {
	conf, ok := logConfMap[name]
	if !ok {
		logConfMap[name] = &QLogConfig{
			Level:      DebugLevel,
			MaxSize:    1,
			MaxAge:     30,
			MaxBackups: 5,
			Compress:   false,
		}
		return logConfMap[name]
	}
	return conf
}
