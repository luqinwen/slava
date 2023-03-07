package lock

import (
	"sort"
	"sync"
)

// Lock结构体为key提供读写锁
type Locks struct {
	table []*sync.RWMutex
}

// 创建一个lock map
func Make(tableSize int) *Locks {
	table := make([]*sync.RWMutex, tableSize)
	for i := 0; i < len(table); i++ {
		table[i] = &sync.RWMutex{}
	}
	return &Locks{table: table}
}

// 设置哈希函数
const ( // 用于哈希函数计算
	prime32 = uint32(16777619)
)

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}

// 找到hash值对应table的索引
func (locks *Locks) spread(hashCode uint32) uint32 {
	if locks == nil {
		panic("dict is nil")
	}
	tableSize := uint32(len(locks.table))
	return (tableSize - 1) & hashCode
}

// 获取某个key的写锁
func (locks *Locks) Lock(key string) {
	index := locks.spread(fnv32(key))
	mu := locks.table[index]
	mu.Lock()
}

// 获取key的读锁
func (locks *Locks) RLock(key string) {
	index := locks.spread(fnv32(key))
	mu := locks.table[index]
	mu.RLock()
}

// 释放key的写锁
func (locks *Locks) UnLock(key string) {
	index := locks.spread(fnv32(key))
	mu := locks.table[index]
	mu.Unlock()
}

// 释放key对应的读锁
func (locks *Locks) RUnLock(key string) {
	index := locks.spread(fnv32(key))
	mu := locks.table[index]
	mu.RUnlock()
}

// 锁定多个key，为了防止多个协程之间循环等待，让所有饿协程都按照一定的顺序获取keys的锁
// 例如，协程A想获得a和b的锁，此时协程A已经拥有a的锁，想获得b的锁，但是协程B也想获得a和b的锁
// 此时协程B已经获取了b的锁，等待a的锁，这种情况下就会出现相互等待对方资源释放而造成死锁的现象。
// 解决的办法，就是按照一定的的顺序进行加锁

// 获取多个keys所在的槽，并且根据reverse字段选择进行排序，如果reverse为true则按照从大到小排序，如果为false则按照从小到大排序
func (locks *Locks) toLockindices(keys []string, reverse bool) []uint32 {
	// 用来存储keys所对应的所有索引
	indexMap := make(map[uint32]struct{})
	for _, key := range keys {
		index := locks.spread(fnv32(key))
		indexMap[index] = struct{}{}
	}
	// 预设容量，这一提升性能，避免底层频繁的扩容
	indices := make([]uint32, 0, len(indexMap))
	for index := range indexMap {
		indices = append(indices, index)
	}
	sort.Slice(indices, func(i, j int) bool {
		if !reverse {
			return indices[i] < indices[j]
		}
		return indices[i] > indices[j]
	})
	return indices
}

// 对多个key进行上锁，按照从小到达的规则进行上锁，主要是为了防止协程相互等待对方的资源而导致死锁
// 这里对多个key值进行上写锁

func (locks *Locks) Locks(keys ...string) {
	indices := locks.toLockindices(keys, false)
	for _, index := range indices {
		mu := locks.table[index]
		mu.Lock()
	}
}

// 对多个key上读锁，防止协程相互等待对方的资源而导致死锁
func (locks *Locks) RLocks(keys ...string) {
	indices := locks.toLockindices(keys, false)
	for _, index := range indices {
		mu := locks.table[index]
		mu.RLock()
	}
}

// 解开多个写锁，解开写锁的时候，按照从大到小的顺序解开
// 先从大的的锁开始解开，上锁从小的锁开始上锁，这样可以防止死锁
// 例如，A协程含有所有的资源锁a，b，c，此时B协程想要上锁，如果A协程先从小的锁开始释放的话
// 这时候A协程释放了a资源，这时候B协程获取了a资源，还要等待b，c 资源，然而这是A协程要处理某些情况又需要a资源
// 这时候A协程就会等待B协程释放a资源，而B协程等待A协程释放b，c资源，这样就形成相互等待对方的资源释放，而形成死锁。
func (locks *Locks) UnLocks(keys ...string) {
	indices := locks.toLockindices(keys, true)
	for _, index := range indices {
		mu := locks.table[index]
		mu.Unlock()
	}
}

// 释放多个key的读锁
func (locks *Locks) RUnLocks(keys ...string) {
	indices := locks.toLockindices(keys, true)
	for _, index := range indices {
		mu := locks.table[index]
		mu.RUnlock()
	}
}

// 对写和读的操作进行上锁，在写和读的命令中允许存在重复的key
func (locks *Locks) RWLocks(writeKeys []string, readKeys []string) {
	keys := append(writeKeys, readKeys...)
	indices := locks.toLockindices(keys, false)

	// 记录writeKeys中的key值的索引
	writeKeysSet := make(map[uint32]struct{}, len(writeKeys))
	for _, key := range writeKeys {
		index := locks.spread(fnv32(key))
		writeKeysSet[index] = struct{}{}
	}
	for _, index := range indices {
		_, ok := writeKeysSet[index]
		mu := locks.table[index]
		if ok {
			mu.Lock()
		} else {
			mu.RLock()
		}
	}
}

// 释放读写操作的锁，写和读的key可能会存在重复

func (locks *Locks) RWUnLocks(writeKeys []string, readKeys []string) {
	keys := append(writeKeys, readKeys...)
	indices := locks.toLockindices(keys, true)
	writeKeysSet := make(map[uint32]struct{}, len(writeKeys))
	for _, key := range writeKeys {
		index := locks.spread(fnv32(key))
		writeKeysSet[index] = struct{}{}
	}
	for _, index := range indices {
		_, ok := writeKeysSet[index]
		mu := locks.table[index]
		if ok {
			mu.Unlock()
		} else {
			mu.RUnlock()
		}
	}
}
