package pool

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

type FakeConn struct {
	mu     sync.Mutex
	closed bool
}

func (c *FakeConn) Close() {
	c.mu.Lock()
	c.closed = true
	time.Sleep(50 * time.Millisecond) // 模拟关闭连接的延迟
	c.mu.Unlock()
}

func NewFakeConn() (interface{}, error) {
	time.Sleep(50 * time.Millisecond) // 模拟创建连接的延迟
	return &FakeConn{}, nil
}

func DestroyFakeConn(x interface{}) {
	c := x.(*FakeConn)
	c.Close()
}

func ProcessWithPool(pool *Pool) {
	conn, err := pool.Get()
	if err != nil {
		fmt.Println("Error getting connection from pool:", err)
		return
	}

	defer pool.Put(conn)

	// 模拟处理任务
	time.Sleep(10 * time.Millisecond)
}

func ProcessWithoutPool() {
	conn, err := NewFakeConn()
	if err != nil {
		fmt.Println("Error creating connection:", err)
		return
	}

	defer DestroyFakeConn(conn)

	// 模拟处理任务
	time.Sleep(10 * time.Millisecond)
}

func BenchmarkWithPool(b *testing.B) {
	cfg := Config{
		MaxIdle:   10,
		MaxActive: 50,
	}

	pool := New(NewFakeConn, DestroyFakeConn, cfg)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ProcessWithPool(pool)
		}
	})
}

func BenchmarkWithoutPool(b *testing.B) {
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			ProcessWithoutPool()
		}
	})
}
