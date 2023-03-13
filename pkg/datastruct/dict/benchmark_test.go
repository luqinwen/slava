package dict

import (
	"math/rand"
	"strconv"
	"sync"
	"testing"

	"slava/internal/data"
)

// 读与写各自操作数
var times = 10000000

// BenchmarkTestConcurrentMap 测试ConcurrentMap
func BenchmarkTestConcurrentMap(b *testing.B) {
	for k := 0; k < b.N; k++ {
		b.StopTimer()
		// 产生10000个不重复的键值对(string -> int)
		testKV := map[string]int{}
		for i := 0; i < 10000; i++ {
			testKV[strconv.Itoa(i)] = i
		}

		// 新建一个ConcurrentMap
		pMap := MakeConcurrent(data.DataDictSize)

		// set到map中
		for k, v := range testKV {
			pMap.Put(k, v)
		}

		// 开始计时
		b.StartTimer()

		wg := sync.WaitGroup{}
		wg.Add(2)

		// 赋值
		go func() {
			// 对随机key,赋值times次
			for i := 0; i < times; i++ {
				index := rand.Intn(times)
				pMap.Put(strconv.Itoa(index), index+1)
			}
			wg.Done()
		}()

		// 读取
		go func() {
			// 对随机key,读取times次
			for i := 0; i < times; i++ {
				index := rand.Intn(times)
				pMap.Get(strconv.Itoa(index))
			}
			wg.Done()
		}()

		// 追加
		go func() {
			for i := times; i < 2*times; i++ {
				index := rand.Intn(times)
				pMap.Put(strconv.Itoa(index), i)
			}
			wg.Done()
		}()

		// 等待两个协程处理完毕
		wg.Wait()
	}
}

// BenchmarkTestSyncMap 测试sync.map
func BenchmarkTestSyncMap(b *testing.B) {
	for k := 0; k < b.N; k++ {
		b.StopTimer()
		// 产生10000个不重复的键值对(string -> int)
		testKV := map[string]int{}
		for i := 0; i < 10000; i++ {
			testKV[strconv.Itoa(i)] = i
		}

		// 新建一个sync.Map
		pMap := &sync.Map{}

		// set到map中
		for k, v := range testKV {
			pMap.Store(k, v)
		}

		// 开始计时
		b.StartTimer()

		wg := sync.WaitGroup{}
		wg.Add(2)

		// 赋值
		go func() {
			// 对随机key,赋值
			for i := 0; i < times; i++ {
				index := rand.Intn(times)
				pMap.Store(strconv.Itoa(index), index+1)
			}
			wg.Done()
		}()

		// 读取
		go func() {
			// 对随机key,读取10万次
			for i := 0; i < times; i++ {
				index := rand.Intn(times)
				pMap.Load(strconv.Itoa(index))
			}
			wg.Done()
		}()

		// 追加
		go func() {
			for i := times; i < 2*times; i++ {
				index := rand.Intn(times)
				pMap.Store(strconv.Itoa(index), i)
			}
			wg.Done()
		}()

		// 等待两个协程处理完毕
		wg.Wait()
	}
}
