package logger

import (
	"context"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type GormZapLogger struct {
	ZapLogger                 *zap.Logger
	LogLevel                  logger.LogLevel
	SlowThreshold             time.Duration // 慢查询阈值
	IgnoreRecordNotFoundError bool          // 是否忽略 ErrRecordNotFound 错误
}

func NewGormZap(zapLogger *zap.Logger) logger.Interface {
	return &GormZapLogger{
		ZapLogger:                 zapLogger,
		LogLevel:                  logger.Info, // 默认记录所有 SQL
		SlowThreshold:             200 * time.Millisecond,
		IgnoreRecordNotFoundError: true,
	}
}

// LogMode 实现 logger.Interface 接口
func (l *GormZapLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *GormZapLogger) Info(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Info {
		l.ZapLogger.Info(msg, zap.Any("data", data))
	}
}

func (l *GormZapLogger) Warn(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Warn {
		l.ZapLogger.Warn(msg, zap.Any("data", data))
	}
}

func (l *GormZapLogger) Error(ctx context.Context, msg string, data ...interface{}) {
	if l.LogLevel >= logger.Error {
		l.ZapLogger.Error(msg, zap.Any("data", data))
	}
}

// Trace 是打印 SQL 的核心入口
func (l *GormZapLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	sql, rows := fc()

	// 1. 安全提取 TraceID
	var traceID string
	if val, ok := ctx.Value("requestID").(string); ok {
		traceID = val
	}

	// 预置基础字段
	fields := []zap.Field{
		zap.Duration("elapsed", elapsed),
		zap.Int64("rows", rows),
		zap.String("sql", sql),
	}

	if traceID != "" {
		fields = append(fields, zap.String("requestID", traceID))
	}

	// 1. 错误处理
	if err != nil && l.LogLevel >= logger.Error {
		if errors.Is(err, gorm.ErrRecordNotFound) && l.IgnoreRecordNotFoundError {
			l.ZapLogger.Debug("Database Record Not Found", fields...)
		} else {
			l.ZapLogger.Error("Database Error", append(fields, zap.Error(err))...)
		}
		return
	}

	// 2. 慢查询处理
	if l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= logger.Warn {
		l.ZapLogger.Warn("Slow SQL Warning", fields...)
		return
	}

	// 3. 正常 SQL 记录
	if l.LogLevel >= logger.Info {
		l.ZapLogger.Info("Database Query", fields...)
	}
}
