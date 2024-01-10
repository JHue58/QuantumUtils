package qnet

import (
	"QuantumUtils/errors"
	"net"
)

// GetListen 获取普通TCP服务端连接
func GetListen(address string) (net.Listener, *errors.QError) {
	listen, err := net.Listen("tcp", address)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return listen, nil
}

// GetDial 获取普通TCP客户端连接
func GetDial(address string) (net.Conn, *errors.QError) {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return nil, errors.New(err.Error())
	}
	return conn, nil
}
