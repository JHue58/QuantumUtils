package v1

import (
	"QuantumUtils/errors"
	"QuantumUtils/qnet/qc"
	"bytes"
	"encoding/binary"
	"io"
)

type QTPBaseParser struct {
	Version uint8
}

// 解析单个消息
func (p QTPBaseParser) parseMsg(buf []byte) *qc.QTPData {

	msg := qc.QTPData{
		Data: buf[qc.HeaderLength:],
	}

	header, err := p.parseHeader(buf)
	if err != nil {
		msg.ParserError = err
		return &msg
	}

	msg.Data = buf[qc.HeaderLength : qc.HeaderLength+header.DataLength]
	msg.ParserError = nil
	msg.Header = *header
	return &msg

}

// 解析消息头
func (p QTPBaseParser) parseHeader(buf []byte) (*qc.QTPHeader, *errors.QError) {
	header := qc.QTPHeader{
		HeaderFlag: qc.HeaderFlag,
		Version:    p.Version,
	}
	if len(buf) < qc.HeaderLength {
		err := errors.New("解析失败，byte切片长度不足，非QMsg协议类型")
		return nil, err
	}

	if !bytes.Equal(buf[:5], qc.HeaderFlag[:]) {
		err := errors.New("解析失败，协议Flag解析失败，非QMsg协议类型")
		return nil, err
	}

	if buf[5] != p.Version {
		err := errors.New("解析失败，协议版本与解析器版本不符合")
		return nil, err
	}

	header.Seq = binary.LittleEndian.Uint64(buf[6:14])

	switch buf[14] {
	case byte(qc.JSON):
		{
			header.Encode = qc.JSON
			break
		}
	case byte(qc.BINARY):
		{
			header.Encode = qc.BINARY
			break
		}
	default:
		{
			err := errors.New("解析失败，数据编码类型解析失败")
			return nil, err
		}

	}

	switch buf[15] {
	case byte(qc.ACK):
		{
			header.MsgType = qc.ACK
			break
		}
	case byte(qc.DATA):
		{
			header.MsgType = qc.DATA
			break
		}
	case byte(qc.RETRY):
		{
			header.MsgType = qc.RETRY
			break
		}
	default:
		{
			err := errors.New("解析失败，消息类型解析失败")
			return nil, err
		}

	}

	switch buf[16] {
	case byte(qc.NoACK):
		{
			header.ACKType = qc.NoACK
			break
		}
	case byte(qc.SyncACK):
		{
			header.ACKType = qc.SyncACK
			break
		}
	case byte(qc.AsyncACK):
		{
			header.ACKType = qc.AsyncACK
			break
		}
	default:
		{
			err := errors.New("解析失败，ACK机制解析失败")
			return nil, err
		}

	}

	header.DataLength = binary.LittleEndian.Uint32(buf[17:21])
	header.Reserved = binary.LittleEndian.Uint16(buf[21:qc.HeaderLength])
	return &header, nil
}

// 分割数据包
func (p QTPBaseParser) splitPackets(buf []byte) [][]byte {
	var packets [][]byte

	// 搜索消息头的位置
	headerPos := bytes.Index(buf, qc.HeaderFlag[:])

	for headerPos != -1 {

		// 获取下一个消息头的位置
		headerPosNext := bytes.Index(buf[headerPos+qc.HeaderLength:], qc.HeaderFlag[:])
		if headerPosNext != -1 {
			// 截取两个消息头之间的数据并加入结果
			packets = append(packets, buf[headerPos:headerPosNext])
			// 将buf更新为第二个消息头后的buf
			buf = buf[headerPosNext:]
			headerPos = headerPosNext
		} else {
			// buf只剩一个消息头，直接截取消息头之后的所有消息加入结果
			packets = append(packets, buf[headerPos:])
			break
		}

	}

	return packets
}

func (p QTPBaseParser) Parse(buf []byte) []*qc.QTPData {
	packets := p.splitPackets(buf)
	var msgSlice = make([]*qc.QTPData, len(packets))
	for index, packet := range packets {
		msg := p.parseMsg(packet)
		msgSlice[index] = msg
	}
	return msgSlice

}

var lastRead *qc.QTPData = nil

func (p QTPBaseParser) ParseReader(reader io.Reader) *qc.QTPData {
	qtpData := qc.QTPData{}
	defer func() { lastRead = &qtpData }()
	headerBuf := make([]byte, qc.HeaderLength)
	read, err := reader.Read(headerBuf)
	if err != nil {
		qtpData.ParserError = errors.New(err.Error())
		return &qtpData
	}

	if read < qc.HeaderLength {

		qtpData.ParserError = errors.New("数据格式不安全，不符合QTP规范，请求断开客户端连接")
		return &qtpData
	}

	header, qErr := p.parseHeader(headerBuf)
	if qErr != nil {
		qErr.WithMessage("数据格式不安全，不符合QTP规范，请求断开客户端连接")
		qtpData.ParserError = qErr
		return &qtpData
	}
	// 如果header中数据长度为0，则不需要继续在read数据，防止阻塞
	if header.DataLength == 0 {
		qtpData.Header = *header
		qtpData.Data = make([]byte, 0)
		return &qtpData
	}

	dataBuf := make([]byte, header.DataLength)
	read, err = reader.Read(dataBuf)

	if err != nil {
		qtpData.ParserError = errors.New(err.Error())
		return &qtpData
	}
	if uint32(read) != header.DataLength {
		qtpData.ParserError = errors.New("数据格式不安全，不符合QTP规范，请求断开客户端连接")
		return &qtpData
	}
	qtpData.Header = *header
	qtpData.Data = dataBuf
	return &qtpData

}

func NewQMsgParser() qc.QTPParser {
	return QTPBaseParser{Version: 1}
}
