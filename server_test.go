package QuantumUtils

import (
	"QuantumUtils/errors"
	"QuantumUtils/qnet"
	"QuantumUtils/qnet/qc"
	"QuantumUtils/qnet/qio"
	"QuantumUtils/qnet/receive"
	"QuantumUtils/qnet/send"
	"fmt"
	"strconv"
	"sync"
	"testing"
	"time"
)

type Counter struct {
	mu    sync.Mutex
	count int
}

func (c *Counter) Add() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.count++
}

func TestServer(t *testing.T) {
	var wg sync.WaitGroup
	cb := receive.NewCallBackHandler()

	count := Counter{
		mu:    sync.Mutex{},
		count: 0,
	}
	mu := sync.Mutex{}
	cb.NoACK(func(sender *send.QTPSender, data *qc.QTPData) {
		//time.Sleep(3 * time.Second)
		//fmt.Println(data)

		defer mu.Unlock()
		mu.Lock()
		//fmt.Println(string(data.Data))
		count.Add()
	})
	cb.SyncACK(func(sender *send.QTPSender, data *qc.QTPData) {
		//time.Sleep(3 * time.Second)
		//fmt.Println(data)

		defer mu.Unlock()
		mu.Lock()
		//fmt.Println(string(data.Data))
		count.Add()
	})
	cb.AsyncACK(func(sender *send.QTPSender, data *qc.QTPData) {
		//time.Sleep(3 * time.Second)
		//fmt.Println(data)

		defer mu.Unlock()
		mu.Lock()
		//fmt.Println(string(data.Data))
		count.Add()
	})
	cb.ConnClosed(func(err *errors.QError) {
		fmt.Println(count.count)
		fmt.Println("连接释放！")
	})
	cb.ConnInit(func(sender *send.QTPSender) {
		sender.SendSyncACK([]byte("确认一下"), qc.BINARY, func(seq uint64, err *errors.QError) {
			if err != nil {
				return
			}
			fmt.Println(strconv.FormatUint(seq, 10) + "确认过眼神~")
		})
		fmt.Println("新连接！")
	})
	wg.Add(1)
	go func() {
		defer wg.Done()
		server, err := qnet.NewQuantumServer("127.0.0.1:8081", cb, qio.QTPWriterConfigDefault())
		if err != nil {
			err.Print()
		}
		server.Start()
	}()

	wg.Wait()
}

func TestClient(t *testing.T) {
	cb := receive.NewCallBackHandler()
	cb.SyncACK(func(sender *send.QTPSender, data *qc.QTPData) {
		str := string(data.Data)
		fmt.Println(str)
		if str == "确认一下" {
			fmt.Println("好的我确认")
		}
	})

	sender, err := qnet.NewQuantumClient("127.0.0.1:8081", send.QTPSenderConfigDefault(), qio.QTPWriterConfigDefault(), cb)
	if err != nil {
		err.Print()
		return
	}
	count := Counter{
		mu:    sync.Mutex{},
		count: 0,
	}

	// 记录开始时间
	startTime := time.Now()
	sendCount := 100000
	//mu := sync.Mutex{}
	for i := 0; i < sendCount; i++ {
		x := i
		d := []byte(strconv.Itoa(x) + " finish")
		lce := func(seq uint64, err *errors.QError) {
			if err == nil {
				count.Add()
			} else {
				err.Print()
			}
			//mu.Lock()
			//defer mu.Unlock()
			//fmt.Println(strconv.Itoa(x) + " finish")
		}
		sender.SendSyncACK(d, qc.BINARY, lce)
	}
	sender.Close()
	// 记录结束时间
	endTime := time.Now()
	// 计算代码执行时间
	elapsedTime := endTime.Sub(startTime)

	// 提取秒数
	elapsedSeconds := elapsedTime.Seconds()

	// 打印耗时（以秒为单位）
	fmt.Printf("代码执行耗时: %.2f秒\n", elapsedSeconds)

}

func TestThreeClient(t *testing.T) {
	cb := receive.NewCallBackHandler()
	cb.SyncACK(func(sender *send.QTPSender, data *qc.QTPData) {
		str := string(data.Data)
		fmt.Println(str)
		if str == "确认一下" {
			fmt.Println("好的我确认")
		}
	})
	wc := qio.QTPWriterConfigDefault()
	wc.SendCap = 1000000
	sender1, _ := qnet.NewQuantumClient("127.0.0.1:8081", send.QTPSenderConfigDefault(), wc, cb)
	sender2, _ := qnet.NewQuantumClient("127.0.0.1:8081", send.QTPSenderConfigDefault(), wc, cb)
	sender3, _ := qnet.NewQuantumClient("127.0.0.1:8081", send.QTPSenderConfigDefault(), wc, cb)

	// 记录开始时间
	startTime := time.Now()
	sendCount := 1000000
	lce := func(seq uint64, err *errors.QError) {
		if err != nil {
			err.Print()
		}
	}
	go sends(sender1, []byte("66finish"), qc.BINARY, lce, qc.NoACK, sendCount)
	go sends(sender2, []byte("66finish"), qc.BINARY, lce, qc.AsyncACK, sendCount)
	go sends(sender3, []byte("66finish"), qc.BINARY, lce, qc.SyncACK, sendCount)
	time.Sleep(3 * time.Second)
	sender1.Close()
	sender2.Close()
	sender3.Close()
	// 记录结束时间
	endTime := time.Now()
	// 计算代码执行时间
	elapsedTime := endTime.Sub(startTime)

	// 提取秒数
	elapsedSeconds := elapsedTime.Seconds()

	// 打印耗时（以秒为单位）
	fmt.Printf("代码执行耗时: %.2f秒\n", elapsedSeconds)

}

func sends(sender *send.QTPSender, data []byte, encode qc.Encode, lce qio.LCE, ackType qc.ACKType, count int) {
	switch ackType {
	case qc.NoACK:
		{
			for i := 0; i < count; i++ {
				sender.SendNoACK(data, encode, lce)
			}
			break
		}
	case qc.SyncACK:
		{
			for i := 0; i < count; i++ {
				sender.SendSyncACK(data, encode, lce)
			}
			break
		}
	case qc.AsyncACK:
		{
			for i := 0; i < count; i++ {
				sender.SendAsyncACK(data, encode, lce)
			}
			break
		}

	}
}
