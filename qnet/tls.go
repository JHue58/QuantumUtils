package qnet

import (
	"QuantumUtils/errors"
	"crypto/tls"
	"net"
)

// 获取安全的TLS服务端连接
func getTlsListen(address string, cerFile string, keyFile string) (net.Listener, *errors.QError) {
	cer, err := tls.LoadX509KeyPair(cerFile, keyFile)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	config := &tls.Config{Certificates: []tls.Certificate{cer}}
	listen, err := tls.Listen("tcp", address, config)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return listen, nil
}
