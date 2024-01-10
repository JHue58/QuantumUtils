package send

import (
	"QuantumUtils/errors"
	"QuantumUtils/goroutine"
	"QuantumUtils/logger"
	"QuantumUtils/qnet/connect"
	"QuantumUtils/qnet/qc"
	v1 "QuantumUtils/qnet/qc/v1"
	"QuantumUtils/qnet/qio"
	"QuantumUtils/qnet/seq"
	"time"
)

// QTPSenderConfig QTPSender配置
type QTPSenderConfig struct {
	// 最大重连次数
	ReconnectAttempts int
}

// QTPSenderConfigDefault Get默认QTPSender配置
func QTPSenderConfigDefault() QTPSenderConfig {
	return QTPSenderConfig{
		ReconnectAttempts: 5,
	}
}

type closedFlag struct {
	sendRHClosed bool
	sendHClosed  bool
	ackHClosed   bool
}

type QTPSender struct {
	connect.QTPConnAccessor
	logger.QLoggerAccessor
	qio.QTPReaderAccessor
	goroutine.GoManagerAccessor
	qio.QTPWriterAccessor
	// QTP包装器
	encoder qc.QTPEncoder
	// 发送请求通道
	sqc    chan *qio.SendReq
	closed closedFlag
}

func (sender *QTPSender) send(sr *qio.SendReq) {
	if sender.closed.sendRHClosed {
		go sr.LCE(0, errors.New("连接已主动关闭，无法提交发送请求"))
	}
	// 提交发送请求
	sender.sqc <- sr
}

func (sender *QTPSender) sendRequestHandle() {
	for {
		select {
		case sr, ok := <-sender.sqc:
			{
				if !ok {
					return
				}
				// 包装数据
				seqN, data, err := sender.encoder.Encode(sr.Data, sr.Config)

				if err != nil {
					go sr.LCE(seqN, err)
					continue
				}
				sr.SeqN = seqN
				sr.Data = data
				// 提交发送
				sender.GetQTPWriter().SendChan <- sr
				// 如果是同步确认消息，则等待消息的生命周期完成
				if sr.Config.ACKType == qc.SyncACK {
					sender.GetQTPWriter().WaitMsgLCE(seqN)
				}
			}
		// 若100秒内无请求
		case <-time.After(time.Millisecond * 100):
			{
				if sender.closed.sendRHClosed {
					return
				}
			}
		}
	}
}

// SendNoACK 发送NoACK消息
func (sender *QTPSender) SendNoACK(data []byte, encode qc.Encode, lce qio.LCE) {
	conf := qc.QTPConfig{
		Encode:  encode,
		MsgType: qc.DATA,
		ACKType: qc.NoACK,
	}
	sr := &qio.SendReq{
		Data:   data,
		Config: conf,
		LCE:    lce,
	}
	sender.send(sr)
}

// SendSyncACK 发送SyncACK消息
func (sender *QTPSender) SendSyncACK(data []byte, encode qc.Encode, lce qio.LCE) {
	conf := qc.QTPConfig{
		Encode:  encode,
		MsgType: qc.DATA,
		ACKType: qc.SyncACK,
	}
	sr := &qio.SendReq{
		Data:   data,
		Config: conf,
		LCE:    lce,
	}
	sender.send(sr)
}

// SendAsyncACK 发送AsyncACK消息
func (sender *QTPSender) SendAsyncACK(data []byte, encode qc.Encode, lce qio.LCE) {
	conf := qc.QTPConfig{
		Encode:  encode,
		MsgType: qc.DATA,
		ACKType: qc.AsyncACK,
	}
	sr := &qio.SendReq{
		Data:   data,
		Config: conf,
		LCE:    lce,
	}
	sender.send(sr)
}

func (sender *QTPSender) ackHandle() {
	// TODO set持久化defer

	for {
		select {
		case data, ok := <-sender.GetQTPReader().ACKChan:
			{
				if !ok {
					return
				}
				if data.ParserError != nil {
					data.ParserError.WithStack()
					sender.GetLogger().QError(data.ParserError)
					return
				}
				// 只接受ACK消息
				if data.Header.MsgType != qc.ACK {
					continue
				}
				// 获取消息序列号
				ackSeq := data.Header.Seq
				// 完成消息生命周期
				sender.GetQTPWriter().MsgFinish(ackSeq)

			}
		case <-time.After(time.Millisecond * 100):
			{
				if sender.closed.ackHClosed {
					return
				}
			}

		}
	}

}

func (sender *QTPSender) Close() *errors.QError {
	// 停止提交发送请求
	sender.closed.sendRHClosed = true
	// 等待处理剩下的发送请求
	sender.GetGoManager().Wait(sendRequestHandle)
	// 停止提交发送
	sender.closed.sendHClosed = true
	// 等待处理剩下的发送
	sender.GetQTPWriter().Close()
	// 等待剩下的消息生命周期结束
	sender.WaitMsgLCE()
	// 停止接受ACK
	sender.closed.ackHClosed = true
	sender.GetGoManager().Wait(ackHandle)
	// 关闭连接
	err := sender.GetQTPConn().Close()
	if err != nil {
		return errors.New(err.Error())
	}
	return nil
}

var ackHandle = goroutine.Group("ackHandle")
var sendRequestHandle = goroutine.Group("sendRHandle")

// Start 启动
func (sender *QTPSender) Start() {
	goroutine.CheckAndGetters(func() any {
		return sender.GetLogger()
	}, func() any {
		return sender.GetQTPConn()
	}, func() any {
		return sender.GetGoManager()
	}, func() any {
		return sender.GetQTPReader()
	}, func() any {
		return sender.GetQTPWriter()
	})
	sender.GetGoManager().Goroutine(ackHandle, sender.ackHandle)
	sender.GetGoManager().Goroutine(sendRequestHandle, sender.sendRequestHandle)
}

// SendRequest 返回有多少个发送请求等待处理
func (sender *QTPSender) SendRequest() int {
	return len(sender.sqc)
}

// WaitMsgLCE 等待所有消息生命周期结束
func (sender *QTPSender) WaitMsgLCE() {
	sender.GetQTPWriter().WaitALLMsgLCE()
}

// NewSender 创建一个Sender
func NewSender(sendCap int) *QTPSender {

	encoder := v1.NewQMsgEncoder(seq.NewSnowFlakeSeqGenerator())
	sender := &QTPSender{
		encoder: encoder,
	}
	sender.sqc = make(chan *qio.SendReq, sendCap)

	return sender

}
