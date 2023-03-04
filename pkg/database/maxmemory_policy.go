package database

import (
	"math/rand"
	"runtime"
	"slava/internal/interface/database"
	"slava/pkg/datastruct/dict"
	"sync"
	"time"
)

const (
	maxMemoryLruAllKeys = iota
	maxMemoryLfuAllKeys
	maxMemoryLruTtl
	maxMemoryLfuTtl
)

// freeMemoryIfNeeded 进行内存清理如果需要的话
func (server *Server) freeMemoryIfNeeded(keyNumsOneRound int, dbNumsOneRound int) {
	var memUsed uint64
	var memStats runtime.MemStats

	for {
		runtime.ReadMemStats(&memStats)
		//memUsed = memStats.HeapAlloc + memStats.HeapIdle // question
		memUsed = memStats.HeapAlloc

		// 不需要清除，直接return
		if memUsed <= server.maxMemory {
			return
		}

		if dbNumsOneRound > len(server.dbSet) {
			dbNumsOneRound = len(server.dbSet)
		}

		// 挑选需要释放key的db，挑选策略可以更改，目前采用简单的随机方式
		dbIds := randomSelectDB(len(server.dbSet), dbNumsOneRound)
		var wg sync.WaitGroup

		for i := 0; i < dbNumsOneRound; i++ {
			slavaDb, _ := server.selectDB(dbIds[i])

			// 暂时不支持random 只支持lru和lfu
			if server.maxMemoryPolicy == maxMemoryLruAllKeys || server.maxMemoryPolicy == maxMemoryLruTtl {
				wg.Add(1)
				if server.maxMemoryPolicy == maxMemoryLruTtl {
					// ttl
					go func() {
						defer wg.Done()
						approximateLruTtl(slavaDb, keyNumsOneRound)
					}()
				} else {
					go func() {
						defer wg.Done()
						approximateLruAll(slavaDb, keyNumsOneRound)
					}()
				}
			}

			if server.maxMemoryPolicy == maxMemoryLfuAllKeys || server.maxMemoryPolicy == maxMemoryLfuTtl {
				wg.Add(1)
				if server.maxMemoryPolicy == maxMemoryLfuTtl {
					// ttl
					go func() {
						defer wg.Done()
						approximateLfuTtl(slavaDb, keyNumsOneRound)
					}()
				} else {
					func() {
						defer wg.Done()
						approximateLfuTtl(slavaDb, keyNumsOneRound)
					}()
				}
			}
		}

		// 等待该轮多个db的清除操作完成后再循环
		wg.Wait()
	}
}

func randomSelectDB(dbNums int, selectNums int) []int {
	rand.Seed(time.Now().UnixNano()) // 设置随机数种子，使用当前时间戳

	nums := make(map[int]struct{})

	for i := 0; i < selectNums; i++ {
		num := rand.Intn(dbNums)
		if _, ok := nums[num]; !ok {
			nums[num] = struct{}{}
		}
	}

	result := make([]int, selectNums)
	i := 0
	for key := range nums {
		result[i] = key
		i++
	}

	return result
}

func approximateLruAll(slavaDb *DB, nums int) {
	data := slavaDb.data.(*dict.ConcurrentDict)
	keyLruMap := data.RandomDistinctKeysWithTime(nums)

	if len(keyLruMap) < 1 {
		return
	}
	var earliestTimestamp int64
	var earliestKey string
	i := 0
	for key, ts := range keyLruMap {
		ts64 := int64(ts)
		t := time.Unix(ts64, 0)
		if i == 0 || t.Before(time.Unix(earliestTimestamp, 0)) {
			earliestTimestamp = ts64
			earliestKey = key
		}
		i++
	}

	slavaDb.Remove(earliestKey)
	return
}

func approximateLruTtl(slavaDb *DB, nums int) {
	data := slavaDb.data.(*dict.ConcurrentDict)
	ttlMap := slavaDb.ttlMap

	candidateKeys := ttlMap.RandomDistinctKeys(nums)
	if len(candidateKeys) < 1 {
		return
	}

	// len(keyLruMap) 可能小于nums
	var earliestTimestamp int64
	var earliestKey string

	for i, key := range candidateKeys {
		val, ok := data.Get(key)
		if ok {
			ts := int64(val.(*database.DataEntity).Lru)
			t := time.Unix(ts, 0)
			if i == 0 || t.Before(time.Unix(earliestTimestamp, 0)) {
				earliestTimestamp = ts
				earliestKey = key
			}
		}
	}

	slavaDb.Remove(earliestKey)
	return
}

func approximateLfuAll(slavaDb *DB, nums int) {
	data := slavaDb.data.(*dict.ConcurrentDict)
	keyLfuMap := data.RandomDistinctKeysWithCount(nums)

	if len(keyLfuMap) < 1 {
		return
	}
	var minimumCount uint32
	var minimumKey string

	i := 0
	for key, count := range keyLfuMap {
		if i == 0 || count > minimumCount {
			minimumCount = count
			minimumKey = key
		}
		i++
	}

	slavaDb.Remove(minimumKey)
	return
}

func approximateLfuTtl(slavaDb *DB, nums int) {
	data := slavaDb.data.(*dict.ConcurrentDict)
	ttlMap := slavaDb.ttlMap

	candidateKeys := ttlMap.RandomDistinctKeys(nums)
	if len(candidateKeys) < 1 {
		return
	}

	// len(keyLruMap) 可能小于nums
	var minimumCount uint32
	var minimumKey string

	for i, key := range candidateKeys {
		val, ok := data.Get(key)
		if ok {
			count := val.(*database.DataEntity).Lfu
			if i == 0 || count > minimumCount {
				minimumCount = count
				minimumKey = key
			}
		}
	}

	slavaDb.Remove(minimumKey)
	return
}
