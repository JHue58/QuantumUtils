package receive

import (
	"QuantumUtils/errors"
	"QuantumUtils/qnet/qc"
	"QuantumUtils/qnet/send"
)

// CallBackHandler 回调集合
type CallBackHandler struct {
	noACKLoop       []func(sender *send.QTPSender, data *qc.QTPData)
	syncACKLoop     []func(sender *send.QTPSender, data *qc.QTPData)
	asyncACKLoop    []func(sender *send.QTPSender, data *qc.QTPData)
	connectInitLoop []func(sender *send.QTPSender)
	closedLoop      []func(err *errors.QError)
}

// NewCallBackHandler 新建一个回调接收器
func NewCallBackHandler() *CallBackHandler {
	return &CallBackHandler{}
}

// NoACK 添加NoACK回调
func (h *CallBackHandler) NoACK(f func(sender *send.QTPSender, data *qc.QTPData)) {
	h.noACKLoop = append(h.noACKLoop, f)
}

// SyncACK 添加同步确认回调
func (h *CallBackHandler) SyncACK(f func(sender *send.QTPSender, data *qc.QTPData)) {
	h.syncACKLoop = append(h.syncACKLoop, f)
}

// AsyncACK 添加异步确认回调
func (h *CallBackHandler) AsyncACK(f func(sender *send.QTPSender, data *qc.QTPData)) {
	h.asyncACKLoop = append(h.asyncACKLoop, f)
}

// ConnInit 添加连接时回调
func (h *CallBackHandler) ConnInit(f func(sender *send.QTPSender)) {
	h.connectInitLoop = append(h.connectInitLoop, f)
}

// ConnClosed 添加连接关闭时回调
func (h *CallBackHandler) ConnClosed(f func(err *errors.QError)) {
	h.closedLoop = append(h.closedLoop, f)
}

// CallBacker 回调接收器
type CallBacker struct {
	Handler *CallBackHandler
	Sender  *send.QTPSender
}

type CallBackerAccessor struct {
	cb *CallBacker
}

func (c *CallBackerAccessor) GetCallBacker() *CallBacker {
	return c.cb
}
func (c *CallBackerAccessor) SetCallBacker(callBacker *CallBacker) {
	c.cb = callBacker
}
