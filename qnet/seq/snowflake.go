package seq

import (
	"QuantumUtils/logger"
	"QuantumUtils/qnet/qc"
	"fmt"
	"sync"
	"time"
)

type Snowflake struct {
	sync.Mutex                  // 锁
	timestamp    int64          // 时间戳 ，毫秒
	workerid     int64          // 工作节点
	datacenterid int64          // 数据中心机房id
	sequence     int64          // 序列号
	logger       logger.QLogger // 日志对象
}

const (
	epoch             = int64(1702607855000)                           // 设置起始时间(时间戳/毫秒)，有效期69年
	timestampBits     = uint(41)                                       // 时间戳占用位数
	datacenteridBits  = uint(2)                                        // 数据中心id所占位数
	workeridBits      = uint(7)                                        // 机器id所占位数
	sequenceBits      = uint(12)                                       // 序列所占的位数
	timestampMax      = int64(-1 ^ (-1 << timestampBits))              // 时间戳最大值
	datacenteridMax   = int64(-1 ^ (-1 << datacenteridBits))           // 支持的最大数据中心id数量
	workeridMax       = int64(-1 ^ (-1 << workeridBits))               // 支持的最大机器id数量
	sequenceMask      = int64(-1 ^ (-1 << sequenceBits))               // 支持的最大序列id数量
	workeridShift     = sequenceBits                                   // 机器id左移位数
	datacenteridShift = sequenceBits + workeridBits                    // 数据中心id左移位数
	timestampShift    = sequenceBits + workeridBits + datacenteridBits // 时间戳左移位数
)

func (s *Snowflake) NextSeq() uint64 {
	s.Lock()
	defer s.Unlock()
	now := time.Now().UnixNano() / 1000000 // 转毫秒
	if s.timestamp == now {
		// 当同一时间戳（精度：毫秒）下多次生成id会增加序列号
		s.sequence = (s.sequence + 1) & sequenceMask
		if s.sequence == 0 {
			// 如果当前序列超出12bit长度，则需要等待下一毫秒
			// 下一毫秒将使用sequence:0
			for now <= s.timestamp {
				now = time.Now().UnixNano() / 1000000
			}
		}
	} else {
		// 不同时间戳（精度：毫秒）下直接使用序列号：0
		s.sequence = 0
	}
	t := now - epoch
	if t > timestampMax {
		s.logger.Error(fmt.Sprintf("epoch must be between 0 and %d", timestampMax-1))
		return 0
	}
	s.timestamp = now
	r := (t)<<timestampShift | (s.datacenterid << datacenteridShift) | (s.workerid << workeridShift) | (s.sequence)
	return uint64(r)
}

// NewSnowFlakeSeqGenerator 获取雪花算法序列号生成器
func NewSnowFlakeSeqGenerator() qc.QTPSeqGenerator {
	l := logger.GetQLogger("SnowFlake")
	sf := Snowflake{
		workerid:     0,
		datacenterid: 0,
		logger:       l,
	}
	return &sf
}
