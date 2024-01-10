package errors

import (
	"fmt"
	"github.com/pkg/errors"
)

type QError struct {
	error error
}

type ErrorThrower interface {
	Error(err *QError)
}

// New 自定义一个QError异常
func New(message string) *QError {
	return &QError{error: errors.New("QError:" + message)}
}

// 实现error接口
func (qError *QError) Error() string {
	return qError.error.Error()
}

// Wrap 同时附加堆栈和信息
func (qError *QError) Wrap(message string) {
	qError.error = errors.Wrap(qError.error, message)
}

// WithMessage 仅附加信息
func (qError *QError) WithMessage(message string) {
	qError.error = errors.WithMessage(qError.error, message)
}

// WithStack 仅附加堆栈
func (qError *QError) WithStack() {
	qError.error = errors.WithStack(qError.error)
}

// ErrorStackMessage 获取带调用堆栈的Error信息
func (qError *QError) ErrorStackMessage() string {
	return fmt.Sprintf("%+v\n", qError.error)
}

// Print 打印异常信息
func (qError *QError) Print() {
	fmt.Println(qError.ErrorStackMessage())
}
