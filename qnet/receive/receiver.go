package receive

import (
	"QuantumUtils/errors"
	"QuantumUtils/goroutine"
	"QuantumUtils/logger"
	"QuantumUtils/qnet/connect"
	"QuantumUtils/qnet/qc"
	v1 "QuantumUtils/qnet/qc/v1"
	"QuantumUtils/qnet/qio"
	"strconv"
	"time"
)

type seqSeq struct {
	Seq uint64
}

func (s seqSeq) NextSeq() uint64 {
	return s.Seq
}

// QTPReceiver QTP消息接收器
type QTPReceiver struct {
	connect.QTPConnAccessor
	logger.QLoggerAccessor
	qio.QTPReaderAccessor
	qio.QTPWriterAccessor
	CallBackerAccessor
}

func (receiver *QTPReceiver) handleData() {

	for {
		select {
		case qtpData, ok := <-receiver.GetQTPReader().DATAChan:
			{
				if !ok {
					return
				}
				switch qtpData.Header.ACKType {
				case qc.NoACK:
					{
						receiver.noACK(qtpData)
						break
					}
				case qc.SyncACK:
					{
						receiver.syncACK(qtpData)
						break
					}
				case qc.AsyncACK:
					{
						receiver.asyncACK(qtpData)
						break
					}
				}
			}
		case <-time.After(100 * time.Millisecond):
			{
				select {
				case err := <-receiver.GetQTPReader().ErrorChan:
					{
						receiver.GetQTPWriter().Close()
						receiver.connClosed(err)
						return
					}
				default:
					continue

				}
			}

		}
	}
}

func (receiver *QTPReceiver) noACK(data *qc.QTPData) {
	go func() {
		for _, f := range receiver.GetCallBacker().Handler.noACKLoop {
			f(receiver.GetCallBacker().Sender, data)
		}
	}()
}
func (receiver *QTPReceiver) syncACK(data *qc.QTPData) {
	for _, f := range receiver.GetCallBacker().Handler.syncACKLoop {
		f(receiver.GetCallBacker().Sender, data)
	}
	receiver.sendACK(data.Header.Seq)

}

func (receiver *QTPReceiver) asyncACK(data *qc.QTPData) {
	fc := make(chan bool)
	go func() {
		for _, f := range receiver.GetCallBacker().Handler.asyncACKLoop {
			f(receiver.GetCallBacker().Sender, data)
		}
		fc <- true
	}()
	go func() {
		<-fc
		receiver.sendACK(data.Header.Seq)
	}()

}
func (receiver *QTPReceiver) connClosed(err *errors.QError) {
	for _, f := range receiver.GetCallBacker().Handler.closedLoop {
		f(err)
	}
}
func (receiver *QTPReceiver) connInit() {
	for _, f := range receiver.GetCallBacker().Handler.connectInitLoop {
		f(receiver.GetCallBacker().Sender)
	}
}

// ack消息的配置
var ackConf = qc.QTPConfig{
	Encode:  qc.BINARY,
	MsgType: qc.ACK,
	ACKType: qc.NoACK,
}

func (receiver *QTPReceiver) ackLCE(dataSeq uint64, err *errors.QError) {
	if err != nil {
		err.WithMessage("ACK消息未成功发送,Seq:" + strconv.FormatUint(dataSeq, 10))
		receiver.GetLogger().Warn(err.ErrorStackMessage())
	}
}

func (receiver *QTPReceiver) sendACK(dataSeq uint64) {
	seqG := seqSeq{Seq: dataSeq}
	encoder := v1.NewQMsgEncoder(&seqG)
	_, ackByte, err := encoder.Encode(make([]byte, 0), ackConf)
	if err != nil {
		err.WithMessage("ACK消息未成功发送,Seq:" + strconv.FormatUint(dataSeq, 10))
		receiver.GetLogger().Warn(err.ErrorStackMessage())
		return
	}
	sendR := &qio.SendReq{
		SeqN:   dataSeq,
		Data:   ackByte,
		Config: ackConf,
		LCE:    receiver.ackLCE,
	}
	receiver.GetQTPWriter().SendChan <- sendR
}

// Start 开启receiver(async)
func (receiver *QTPReceiver) Start() {
	goroutine.CheckAndGetters(func() any {
		return receiver.GetQTPConn()
	}, func() any {
		return receiver.GetLogger()
	},
		func() any {
			return receiver.GetQTPReader()
		},
		func() any {
			return receiver.GetQTPWriter()
		},
		func() any {
			return receiver.GetCallBacker()
		})
	receiver.connInit()

	go receiver.handleData()
}

// NewQTPReceiver 创建一个QTPReceiver
func NewQTPReceiver() *QTPReceiver {
	return &QTPReceiver{}
}
