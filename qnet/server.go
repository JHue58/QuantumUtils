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

type QTPServer struct {
	wConf qio.QTPWriterConfig
	logger.QLoggerAccessor
	callBackH *receive.CallBackHandler
	listener  net.Listener
}

func (server *QTPServer) Start() {
	// 退出时关闭
	defer connect.RegisterClose(server.listener, server.GetLogger().QError)

	for {
		conn, err := server.listener.Accept()
		if err != nil {
			server.GetLogger().Error(err.Error())
			continue
		}

		go server.connHandle(conn)

	}
}

func (server *QTPServer) connHandle(conn net.Conn) {
	// 新建receiver
	newReceiver := receive.NewQTPReceiver()
	// 新建sender
	newSender := send.NewSender(server.wConf.SendCap)
	newCallBacker := &receive.CallBacker{
		Handler: server.callBackH,
		Sender:  newSender,
	}

	newSender.SetLogger(server.GetLogger())
	newReceiver.SetLogger(server.GetLogger())

	newReceiver.SetCallBacker(newCallBacker)

	newReceiver.SetQTPConn(conn)
	newSender.SetQTPConn(conn)

	newReader := qio.NewQTPReader(conn, qio.QTPReaderConfigDefault())
	newReceiver.SetQTPReader(newReader)
	newSender.SetQTPReader(newReader)

	newWriter := qio.NewQTPWriter(conn, server.wConf)
	newReceiver.SetQTPWriter(newWriter)
	newSender.SetQTPWriter(newWriter)

	newGm := &goroutine.GoManager{}
	newSender.SetGoManager(newGm)
	newWriter.SetGoManager(newGm)

	newReader.Start()
	newWriter.Start()
	newReceiver.Start()
	newSender.Start()
}

// NewQuantumServer 创建一个基于普通连接的QuantumServer
func NewQuantumServer(address string, callBackH *receive.CallBackHandler, wCof qio.QTPWriterConfig) (*QTPServer, *errors.QError) {
	listen, err := GetListen(address)
	if err != nil {
		return nil, err
	}
	logger.GetLogConfig("QTPServer").Level = logger.WarnLevel
	qLogger := logger.GetQLogger("QTPServer")
	server := QTPServer{
		callBackH: callBackH,
		listener:  listen,
		wConf:     wCof,
	}
	server.SetLogger(qLogger)

	return &server, nil
}
