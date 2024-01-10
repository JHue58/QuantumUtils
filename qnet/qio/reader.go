package qio

import (
	"QuantumUtils/errors"
	"QuantumUtils/goroutine"
	"QuantumUtils/qnet/connect"
	"QuantumUtils/qnet/qc"
	v1 "QuantumUtils/qnet/qc/v1"
)

// QTPReaderAccessor QTPReader存取器
type QTPReaderAccessor struct {
	reader *QTPReader
}

func (q *QTPReaderAccessor) GetQTPReader() *QTPReader {
	return q.reader
}
func (q *QTPReaderAccessor) SetQTPReader(r *QTPReader) {
	q.reader = r
}

// QTPReader 从连接中持续读取QTP数据
type QTPReader struct {
	connect.QTPConnAccessor
	// DATA以及RETRY消息类型的通道
	DATAChan chan *qc.QTPData
	// ACK消息类型通道
	ACKChan chan *qc.QTPData
	// 断连后的错误消息通道
	ErrorChan chan *errors.QError
}

type QTPReaderConfig struct {
	// Data消息缓存长度
	DataCap int
	// ACK消息缓存长度
	ACKCap int
}

// 初始化通道
func (r *QTPReader) initChan(conf QTPReaderConfig) {
	r.DATAChan = make(chan *qc.QTPData, conf.DataCap)
	r.ACKChan = make(chan *qc.QTPData, conf.ACKCap)
	r.ErrorChan = make(chan *errors.QError)
}

func (r *QTPReader) start() {
	parser := v1.NewQMsgParser()
	for {
		qtpData := parser.ParseReader(r.GetQTPConn())
		if qtpData.ParserError != nil {
			// reader解析错误
			//receiver.serverError(qtpData.ParserError)
			r.ErrorChan <- qtpData.ParserError
			break
		}
		switch qtpData.Header.MsgType {
		case qc.ACK:
			{

				r.ACKChan <- qtpData
				break
			}
		case qc.DATA:
			{
				r.DATAChan <- qtpData
				break
			}
		case qc.RETRY:
			{
				r.DATAChan <- qtpData
				break
			}

		}

	}
}

// Start 启动
func (r *QTPReader) Start() {
	goroutine.CheckAndGetters(func() any {
		return r.GetQTPConn()
	})
	go r.start()
}

// NewQTPReader 新建QTPReader
func NewQTPReader(conn connect.QTPConn, conf QTPReaderConfig) *QTPReader {
	reader := QTPReader{}
	reader.SetQTPConn(conn)
	reader.initChan(conf)
	return &reader
}

// QTPReaderConfigDefault 默认配置
func QTPReaderConfigDefault() QTPReaderConfig {
	return QTPReaderConfig{
		DataCap: 1000,
		ACKCap:  1000,
	}
}
