package qio

import (
	"QuantumUtils/errors"
	"QuantumUtils/goroutine"
	"QuantumUtils/qnet/connect"
	"QuantumUtils/qnet/qc"
	cmap "github.com/orcaman/concurrent-map/v2"
	"time"
)

var msgLife = goroutine.Group("msgLife")
var sendH = goroutine.Group("sendH")

// QTPWriterConfig QTPWriter配置
type QTPWriterConfig struct {
	// 重发消息配置
	RConfig RetryConfig
	// 发送缓冲区长度
	SendCap int
}

// QTPWriterAccessor QTPWriter存取器
type QTPWriterAccessor struct {
	writer *QTPWriter
}

func (q *QTPWriterAccessor) GetQTPWriter() *QTPWriter {
	return q.writer
}
func (q *QTPWriterAccessor) SetQTPWriter(w *QTPWriter) {
	q.writer = w
}

type QTPWriter struct {
	connect.QTPConnAccessor
	goroutine.GoManagerAccessor
	// 消息发送chan
	SendChan chan *SendReq
	// 重试消息集合
	rs retrySet
	// 重发消息配置
	rConfig RetryConfig

	closed bool
}

func (q *QTPWriter) start() {
	for {
		select {
		case sr, ok := <-q.SendChan:
			{
				if !ok {
					// 通道关闭，退出循环
					return
				}
				switch sr.Config.ACKType {
				case qc.NoACK:
					{
						_, err := q.GetQTPConn().Write(sr.Data)
						if err != nil {
							go sr.LCE(sr.SeqN, errors.New(err.Error()))
							continue
						}
						go sr.LCE(sr.SeqN, nil)
						break
					}
				case qc.SyncACK:
					{
						rd := q.rs.append(sr.SeqN, sr.Data, q.resend, sr.LCE)
						_, err := q.GetQTPConn().Write(sr.Data)
						if err != nil {
							go sr.LCE(sr.SeqN, errors.New(err.Error()))
							continue
						}
						// 开启消息生命周期
						q.GetGoManager().Goroutine(msgLife, rd.startLife)
						break
					}
				case qc.AsyncACK:
					{
						rd := q.rs.append(sr.SeqN, sr.Data, q.resend, sr.LCE)
						_, err := q.GetQTPConn().Write(sr.Data)
						if err != nil {
							go sr.LCE(sr.SeqN, errors.New(err.Error()))
							continue
						}
						// 开启消息生命周期
						q.GetGoManager().Goroutine(msgLife, rd.startLife)
						break
					}

				}
			}
		case <-time.After(time.Millisecond * 100):
			{
				if q.closed {
					// 如果closed为真，退出循环
					//fmt.Println("send exit")
					return
				}
			}

		}

	}
}

// Close 关闭Writer，该方法会阻塞，等到处理完所有send请求再关闭
func (q *QTPWriter) Close() {
	q.closed = true
	q.GetGoManager().Wait(sendH)
}

func (q *QTPWriter) Start() {
	goroutine.CheckAndGetters(func() any {
		return q.GetQTPConn()
	}, func() any {
		return q.GetGoManager()
	})
	q.GetGoManager().Goroutine(sendH, q.start)
}

// MsgFinish 完成消息的生命周期
func (q *QTPWriter) MsgFinish(seq uint64) {
	q.rs.delete(seq)
}

// WaitALLMsgLCE 等待所有消息生命周期结束
func (q *QTPWriter) WaitALLMsgLCE() {
	q.GetGoManager().Wait(msgLife)
}

// WaitMsgLCE 等待消息生命周期结束
func (q *QTPWriter) WaitMsgLCE(seq uint64) {
	d := q.rs.get(seq)
	if d != nil {
		d.wait()
	}

}

func (q *QTPWriter) resend(data []byte) {
	_, _ = q.GetQTPConn().Write(data)
}

// NewQTPWriter 新建QTPWriter
func NewQTPWriter(conn connect.QTPConn, wConfig QTPWriterConfig) *QTPWriter {
	writer := QTPWriter{}
	writer.SendChan = make(chan *SendReq, wConfig.SendCap)
	writer.rs = retrySet{
		maps:    cmap.New[*retryData](),
		rConfig: wConfig.RConfig,
	}
	writer.SetQTPConn(conn)
	return &writer
}

// SendReq 发送请求
type SendReq struct {
	SeqN   uint64
	Data   []byte
	Config qc.QTPConfig
	// 生命周期结束的回调
	LCE LCE
}

// QTPWriterConfigDefault 默认配置
func QTPWriterConfigDefault() QTPWriterConfig {
	return QTPWriterConfig{
		RConfig: RetryConfigDefault(),
		SendCap: 1000,
	}
}
