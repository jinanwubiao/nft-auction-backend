package logger

import (
	"nft-auction-backend/internal/config"
	"os"
	"path"
	"sync"

	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	logFileName = "server.log"

	consoleMode = "console"
	volumeMode  = "volume"
	bothMode    = "both"

	maxSize   = 100
	maxBackup = 5
)

var (
	logLevel  zapcore.Level
	zapLogger *zap.Logger

	once sync.Once
)

func Init(c config.LogConf) (*zap.Logger, error) {
	if c.KeepDays == 0 {
		c.KeepDays = 7
	}

	setupLogLevel(c)
	var cores []zapcore.Core
	encoder := getEncoder()

	// 只要模式是 console 或 both，就加入控制台输出
	if c.Mode == consoleMode || c.Mode == bothMode {
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), logLevel))
	}
	// 只要模式是 volume、both 或者默认缺省情况，就加入文件输出
	if c.Mode == volumeMode || c.Mode == bothMode || c.Mode == "" {
		cores = append(cores, zapcore.NewCore(encoder, getFileWriter(c), logLevel))
	}
	once.Do(func() {
		zapLogger = zap.New(zapcore.NewTee(cores...), zap.AddCaller())
	})
	return zapLogger, nil
}

func Sync() {
	if zapLogger != nil {
		zapLogger.Sync()
	}
}

// L 返回原生的高性能结构化 Logger
func L() *zap.Logger {
	if zapLogger == nil {
		// 防御性设计：防止未初始化时调用导致进程崩溃
		// 这里返回一个生产环境默认配置的预设 Logger，或者 zap.NewNop()
		return zap.L()
	}
	return zapLogger
}

// S 返回支持格式化输出（如 Debugf）的 SugaredLogger
func S() *zap.SugaredLogger {
	if zapLogger == nil {
		return zap.S() // 返回预设的全局标准 SugarLogger，避免空指针 Panic
	}
	return zapLogger.WithOptions(zap.AddCallerSkip(1)).Sugar()
}

func getConsoleCore() zapcore.Core {
	core := zapcore.NewTee(
		zapcore.NewCore(getEncoder(), zapcore.AddSync(os.Stdout), logLevel),
	)
	return core
}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getFileWriter(c config.LogConf) zapcore.WriteSyncer {
	p := path.Join(c.Path, logFileName)
	lumberJackLogger := &lumberjack.Logger{
		Filename:   p,
		MaxSize:    maxSize,
		MaxBackups: maxBackup,
		MaxAge:     c.KeepDays,
		Compress:   c.Compress,
	}
	return zapcore.AddSync(lumberJackLogger)
}

func setupLogLevel(c config.LogConf) {
	l, err := zapcore.ParseLevel(c.Level)
	if err != nil {
		logLevel = zap.InfoLevel
	} else {
		logLevel = l
	}
}
