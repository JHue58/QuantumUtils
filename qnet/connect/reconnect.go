package connect

import (
	"QuantumUtils/errors"
	"QuantumUtils/logger"
	"net"
	"sync"
	"time"
)

type Reconnecter struct {
	logger.QLoggerAccessor
	reconnectAttempts int                               // 掉线重连次数
	reconnectHandle   func() (net.Conn, *errors.QError) // 重连方法
	lastResult        ReconnecterResult                 // 上一次重连结果
	inProgress        bool                              // 是否正在进行重连
	mu                sync.RWMutex                      // 同步锁
	closed            bool
}

func NewReconnecter(attempts int, rcHandle func() (net.Conn, *errors.QError)) *Reconnecter {
	rc := Reconnecter{
		reconnectAttempts: attempts,
		reconnectHandle:   rcHandle,
		lastResult:        ReconnecterResult{},
		inProgress:        false,
		mu:                sync.RWMutex{},
	}
	return &rc
}

type ReconnecterResult struct {
	conn net.Conn
	err  *errors.QError
}

func (r *Reconnecter) Close() {
	r.closed = true
}

// TryReconnect 尝试重连
// 在多个协程同时请求时只有一个协程在执行重连
// 重连时会被阻塞
// 重连成功时能获得统一的conn对象
// 当重连对象已主动关闭时，返回两个nil值
func (r *Reconnecter) TryReconnect() (net.Conn, *errors.QError) {
	defer func() {
		if r.lastResult.err != nil {
			r.Close()
		}
	}()
	if r.closed {
		return nil, nil
	}
	r.mu.RLock()
	if r.inProgress {
		r.mu.RUnlock()
		// 等待reconnect完成
		for r.inProgress {
		}
		return r.lastResult.conn, r.lastResult.err

	}
	r.mu.RUnlock()

	// 设置inProgress为true，防止其他goroutine进入reconnect
	r.mu.Lock()
	r.inProgress = true
	r.mu.Unlock()

	c := make(chan ReconnecterResult, 1)
	// 执行重连
	go r.reconnect(c)

	r.lastResult = <-c
	// 重置inProgress，允许其他单个goroutine进入reconnect
	r.mu.Lock()
	r.inProgress = false
	r.mu.Unlock()
	return r.lastResult.conn, r.lastResult.err

}

func (r *Reconnecter) reconnect(c chan<- ReconnecterResult) {
	res := ReconnecterResult{}
	defer func() { c <- res }()

	r.GetLogger().Error("服务掉线，正在尝试重新连接")
	count := 0
	for count < r.reconnectAttempts {
		res.conn, res.err = r.reconnectHandle()
		if res.err != nil {
			count++
			time.Sleep(2 * time.Second)
			continue
		}
		r.GetLogger().Info("重连成功")
		return
	}
	r.GetLogger().Error("重新连接失败，超过重连次数")
}
