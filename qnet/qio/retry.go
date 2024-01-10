package qio

import (
	"QuantumUtils/errors"
	"QuantumUtils/qnet/qc"
	cmap "github.com/orcaman/concurrent-map/v2"
	"strconv"
	"sync"
	"time"
)

// LCE 生命周期结束的回调，发送成功时err为nil
type LCE func(seq uint64, err *errors.QError)

// RetryConfig 重发配置
type RetryConfig struct {
	RetryAttempts int           // 消息超时重传次数
	RetryTimeout  time.Duration // 消息超时重传时间
}

// RetryConfigDefault Get默认重发配置
func RetryConfigDefault() RetryConfig {
	return RetryConfig{
		RetryAttempts: 3,
		RetryTimeout:  10 * time.Second,
	}
}

// 重发集合
type retrySet struct {
	maps    cmap.ConcurrentMap[string, *retryData]
	rConfig RetryConfig
}

func (r *retrySet) append(seq uint64, data []byte, retryH func(data []byte), lce LCE) *retryData {
	rData := &retryData{
		encodedData: data,
		retryCount:  0,
		sendTime:    time.Now(),
		retryH:      retryH,
		lce:         lce,
		done:        make(chan bool, 1),
		seq:         seq,
		rConfig:     r.rConfig,
	}
	rData.wg.Add(1)
	r.maps.Set(strconv.FormatUint(seq, 10), rData)
	return rData
}

func (r *retrySet) get(seq uint64) *retryData {
	data, ok := r.maps.Get(strconv.FormatUint(seq, 10))
	if !ok {
		return nil
	}
	return data
}

// 结束消息的生命周期
func (r *retrySet) delete(seq uint64) {
	seqStr := strconv.FormatUint(seq, 10)
	rData, ok := r.maps.Get(seqStr)

	if !ok {
		//fmt.Println("比写早了！")
		return
	}
	r.maps.Remove(seqStr)
	rData.done <- true
}

type retryData struct {
	wg sync.WaitGroup
	// QTP包装后的data
	encodedData []byte
	// 已重试的次数
	retryCount int
	// 该消息的发送时间
	sendTime time.Time
	// 重发的方法
	retryH func(data []byte)
	// LCE
	lce LCE
	// 生命周期结束通知chan
	done chan bool
	// 消息序列号
	seq uint64
	// 重发配置
	rConfig RetryConfig
}

func (r *retryData) startLife() {
	defer r.wg.Done()
	timer := r.getTimeoutTimer()
	select {
	case <-timer.C:
		{
			r.retry()
			return
		}
	case <-r.done:
		{
			timer.Stop()
			go r.lce(r.seq, nil)
			return
		}
	}

}

// 等待该消息的生命周期结束
func (r *retryData) wait() {
	r.wg.Wait()
}

func (r *retryData) getTimeoutTimer() *time.Timer {
	timeSinceSend := time.Since(r.sendTime)
	if timeSinceSend > r.rConfig.RetryTimeout {
		return time.NewTimer(0)
	} else {
		return time.NewTimer(r.rConfig.RetryTimeout - timeSinceSend)
	}
}

func (r *retryData) retry() {
	// 将header改为retry
	r.encodedData[15] = byte(qc.RETRY)
	// retry计数器
	count := 0
	for count < r.rConfig.RetryAttempts {
		// 执行retry方法
		r.retryH(r.encodedData)
		// retry后检测是否收到ACK，2S后没收到则继续retry
		select {
		case <-time.After(time.Second * 2):
			{
				count++
			}
		case <-r.done:
			{
				go r.lce(r.seq, nil)
				return
			}
		}

	}
	go r.lce(r.seq, errors.New("该消息发送失败"))
}
