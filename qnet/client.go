package qnet

import (
	"QuantumUtils/errors"
	"QuantumUtils/goroutine"
	"QuantumUtils/logger"
	"QuantumUtils/qnet/connect"
	"QuantumUtils/qnet/qio"
	"QuantumUtils/qnet/receive"
	"QuantumUtils/qnet/send"
	"net"
)

// NewQuantumClient 新建一个基于普通TCP连接的Client
func NewQuantumClient(address string, sConf send.QTPSenderConfig, wConf qio.QTPWriterConfig, handler *receive.CallBackHandler) (*send.QTPSender, *errors.QError) {
	reconnectHandle := func() (conn net.Conn, err *errors.QError) {
		return GetDial(address)
	}
	conn, err := reconnectHandle()
	if err != nil {
		return nil, err
	}

	logger.GetLogConfig("QTPClient").Level = logger.WarnLevel
	qLogger := logger.GetQLogger("QTPClient")

	gm := &goroutine.GoManager{}
	rc := connect.NewReconnecter(sConf.ReconnectAttempts, reconnectHandle)
	qConn := connect.NewQTPConn(conn, rc)

	sender := send.NewSender(wConf.SendCap)
	receiver := receive.NewQTPReceiver()
	writer := qio.NewQTPWriter(qConn, wConf)
	reader := qio.NewQTPReader(qConn, qio.QTPReaderConfigDefault())

	// 设置conn
	sender.SetQTPConn(qConn)
	receiver.SetQTPConn(qConn)

	// 设置logger
	sender.SetLogger(qLogger)
	rc.SetLogger(qLogger)
	receiver.SetLogger(qLogger)

	// 设置gm
	sender.SetGoManager(gm)
	writer.SetGoManager(gm)

	// 设置CallBacker
	receiver.SetCallBacker(&receive.CallBacker{
		Handler: handler,
		Sender:  sender,
	})

	// 设置reader和writer
	sender.SetQTPReader(reader)
	sender.SetQTPWriter(writer)
	receiver.SetQTPReader(reader)
	receiver.SetQTPWriter(writer)

	// 启动！
	reader.Start()
	receiver.Start()
	writer.Start()
	sender.Start()
	return sender, nil
}
