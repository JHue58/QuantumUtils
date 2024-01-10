package qc

import (
	"QuantumUtils/errors"
	"io"
)

// QTP(Quantum Transport Protocol)为QC(Quantum Communication Protocol)的传输部分，用于规定Byte流的解码方式

// HeaderFlag 消息头Flag
var HeaderFlag = [5]byte{0x78, 0x51, 0x6d, 0x73, 0x67}

// HeaderLength 消息头byte长度
const HeaderLength = 23

// Encode 数据的编码方式
type Encode uint8

// MsgType 该条消息的消息类型
type MsgType uint8

// ACKType ACK机制
type ACKType uint8

const (
	// JSON Json编码
	JSON Encode = iota
	// BINARY 二进制byte流
	BINARY
)

const (
	// ACK ACK消息
	ACK MsgType = iota
	// DATA DATA消息
	DATA
	// RETRY 重发消息
	RETRY
)

const (
	// NoACK 消息无需确认，当消息发送后接收方无需返回ACK信息
	NoACK ACKType = iota
	// SyncACK 同步确认，当消息发送后，接收方接收后需要将这个消息完成业务流程且返回ACK信息后，才可接收新消息，保证消息的可靠性与顺序性
	SyncACK
	// AsyncACK 异步确认，当消息发送后，接收方接收后可继续接收新的消息，且业务流程完成后会返回该消息的ACK，保证消息可靠性，但不保证顺序性
	AsyncACK
)

// QTPHeader QTP协议消息头，共占23byte
type QTPHeader struct {
	// 标志头，占5byte
	HeaderFlag [5]byte
	// 协议版本号，占1byte
	Version uint8
	// 消息序列号，占8byte
	Seq uint64
	// 数据编码类型，占1byte
	Encode Encode
	// 消息类型，占1byte
	MsgType MsgType
	// ACK机制，占1byte
	ACKType ACKType
	// 数据长度，占4byte
	DataLength uint32
	// 预留位置，占2byte
	Reserved uint16
}

type QTPData struct {
	Header QTPHeader
	Data   []byte
	// 若不为nil，则表明该QTPData无法被正常解析
	ParserError *errors.QError
}
type QTPConfig struct {
	Encode  Encode
	MsgType MsgType
	ACKType ACKType
}

// QTPParser QTP协议解析器
type QTPParser interface {
	// Parse 将数据包解析为QTPData，同时解决粘包问题，解析失败的会被丢弃
	Parse(buf []byte) []*QTPData
	// ParseReader 从Reader中将数据包解析为QTPData
	ParseReader(reader io.Reader) *QTPData
}

// QTPEncoder QTP协议包装器
type QTPEncoder interface {
	// Encode 包装数据
	Encode(buf []byte, config QTPConfig) (uint64, []byte, *errors.QError)
}

// QTPSeqGenerator QTP序列号生成器
type QTPSeqGenerator interface {
	NextSeq() uint64
}
