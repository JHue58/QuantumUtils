package v1

import (
	"QuantumUtils/errors"
	"QuantumUtils/qnet/qc"
	"encoding/binary"
	"math"
)

type QTPBaseEncoder struct {
	seqGenerator *qc.QTPSeqGenerator
	version      uint8
}

func NewQMsgEncoder(seqGenerator qc.QTPSeqGenerator) qc.QTPEncoder {
	return &QTPBaseEncoder{
		seqGenerator: &seqGenerator,
		version:      1,
	}
}

func (e *QTPBaseEncoder) Encode(buf []byte, config qc.QTPConfig) (uint64, []byte, *errors.QError) {
	dataLength := len(buf)
	if dataLength > math.MaxUint32 || dataLength < 0 {
		return 0, nil, errors.New("QMsg协议包装失败，数据长度超过了QMsg协议最大支持的4GB")
	}

	var dataLengthBuf [4]byte
	binary.LittleEndian.PutUint32(dataLengthBuf[:], uint32(dataLength))

	var seqBuf [8]byte
	seqG := *e.seqGenerator
	seqN := seqG.NextSeq()
	binary.LittleEndian.PutUint64(seqBuf[:], seqN)

	headerBuf := []byte{
		qc.HeaderFlag[0], qc.HeaderFlag[1], qc.HeaderFlag[2], qc.HeaderFlag[3], qc.HeaderFlag[4], // 消息头Flag
		e.version,                                                                              // 协议版本
		seqBuf[0], seqBuf[1], seqBuf[2], seqBuf[3], seqBuf[4], seqBuf[5], seqBuf[6], seqBuf[7], // 消息序列号
		byte(config.Encode),                                                    // 数据编码类型
		byte(config.MsgType),                                                   // 消息类型
		byte(config.ACKType),                                                   // ACK机制
		dataLengthBuf[0], dataLengthBuf[1], dataLengthBuf[2], dataLengthBuf[3], // 消息数据长度
		0x00, 0x00, // 两个预留字节
	}

	// 把headerBuf插入到buf头部
	result := make([]byte, qc.HeaderLength+dataLength)
	copy(result, headerBuf)
	copy(result[qc.HeaderLength:], buf)
	return seqN, result, nil

}
