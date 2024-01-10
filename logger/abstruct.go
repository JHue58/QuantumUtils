package logger

import "QuantumUtils/errors"

// QLogger QLogger接口，由GetLogger统一返回
type QLogger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
	QError(err *errors.QError)
}

type QLoggerAccessor struct {
	l QLogger
}

func (q *QLoggerAccessor) GetLogger() QLogger {
	return q.l
}
func (q *QLoggerAccessor) SetLogger(l QLogger) {
	q.l = l
}

// 多Logger对象分别管理
var loggerMap = make(map[string]QLogger)

// GetQLogger 无则创建，有则获取QLogger
func GetQLogger(name string) QLogger {
	logger, ok := loggerMap[name]
	if !ok {
		newLogger(name)
		logger = loggerMap[name]
	}
	return logger
}
