package goroutine

import (
	"sync"
	"sync/atomic"
)

type Group string

// GoManager 协程管理器，用于分组管理协程
type GoManager struct {
	wgMap      sync.Map
	counterMap sync.Map
}

type GoManagerAccessor struct {
	gm *GoManager
}

func (g *GoManagerAccessor) GetGoManager() *GoManager {
	return g.gm
}
func (g *GoManagerAccessor) SetGoManager(m *GoManager) {
	g.gm = m
}

type Counter struct {
	count int32
}

func (c *Counter) Add() {
	atomic.AddInt32(&c.count, 1)
}
func (c *Counter) Sub() {
	atomic.AddInt32(&c.count, -1)
}
func (c *Counter) Count() int32 {
	return atomic.LoadInt32(&c.count)
}

// Goroutine 启动协程
func (g *GoManager) Goroutine(group Group, f func()) {
	wg, _ := g.wgMap.LoadOrStore(group, &sync.WaitGroup{})
	counter, _ := g.counterMap.LoadOrStore(group, &Counter{})
	wg.(*sync.WaitGroup).Add(1)
	counter.(*Counter).Add()
	go func() {
		defer func() {
			wg.(*sync.WaitGroup).Done()
			counter.(*Counter).Sub()
		}()
		f()
	}()
}

// Wait 等待某个协程组结束
func (g *GoManager) Wait(group Group) {
	wg, ok := g.wgMap.Load(group)
	if !ok {
		return
	}
	wg.(*sync.WaitGroup).Wait()
	g.wgMap.Delete(group)
	g.counterMap.Delete(group)
}

// Count Get某个协程组的计数
func (g *GoManager) Count(group Group) int {
	count, ok := g.counterMap.Load(group)
	if !ok {
		return 0
	}
	return int(count.(*Counter).Count())
}
