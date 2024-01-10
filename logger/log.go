package logger

import (
	"QuantumUtils/errors"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
)

// 一个基于zap日志库的QLogger实现
type zapQLogger struct {
	name   string
	logger *zap.SugaredLogger
	Conf   *QLogConfig
}

func (z zapQLogger) Debug(msg string) {
	defer z.logger.Sync()
	z.logger.Debugf("%s-Debug:%s", z.name, msg)
}

func (z zapQLogger) Info(msg string) {
	defer z.logger.Sync()
	z.logger.Infof("%s-Info:%s", z.name, msg)
}

func (z zapQLogger) Warn(msg string) {
	defer z.logger.Sync()
	z.logger.Warnf("%s-Warn:%s", z.name, msg)
}

func (z zapQLogger) Error(msg string) {
	defer z.logger.Sync()
	z.logger.Errorf("%s-Error:%s", z.name, msg)
}
func (z zapQLogger) QError(err *errors.QError) {
	//defer z.logger.Sync()
	err.WithStack()
	z.Error(err.ErrorStackMessage())
}

func newLogger(name string) {
	writeSyncer := getLogWriter(name)
	encoder := getEncoder()
	fileCore := zapcore.NewCore(encoder, writeSyncer, GetLogConfig(name).Level)
	consoleCore := zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), GetLogConfig(name).Level)
	teeCore := zapcore.NewTee(fileCore, consoleCore)
	logger := zap.New(teeCore, zap.AddCaller())
	zq := zapQLogger{
		name:   name,
		logger: logger.Sugar(),
		Conf:   GetLogConfig(name),
	}
	loggerMap[name] = zq

}

func getEncoder() zapcore.Encoder {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter(name string) zapcore.WriteSyncer {
	lumberJackLogger := &lumberjack.Logger{
		Filename:   "./log/" + name + "/log.log",
		MaxSize:    GetLogConfig(name).MaxSize,
		MaxBackups: GetLogConfig(name).MaxBackups,
		MaxAge:     GetLogConfig(name).MaxAge,
		Compress:   GetLogConfig(name).Compress,
	}
	return zapcore.AddSync(lumberJackLogger)
}
