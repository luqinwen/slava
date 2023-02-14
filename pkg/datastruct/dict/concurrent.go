package dict

import (
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	. "slava/internal/data"
)

// 实现dict接口

//解决并发读写map的思路是加锁，或者把一个map切分成若干个小map，对key进行哈希。

// 实现内存KV存储数据库将采用分段锁策略，将key分散到固定数量的shard中避免rehash操作。
// shard是有锁保护的map，当shard进行rehash时会阻塞shard内的读写，但是不会对别的shard造成影响

// 不使用sync.Map的原因，sync.Map适用于读多写少的场景
// m.dirty刚被提升后会将m.read复制到新的m.dirty中，在数据量较大的情况下复制操作会阻塞所有协程，存在较大的隐患

type ConcurrentDict struct {
	table      []*shard // 表示所有的map的集合
	count      int32
	shardCount int // shard的数量，即map的数量
}

type shard struct {
	m     map[string]interface{}
	mutex sync.RWMutex
}

func computeCapacity(param int) (size int) {
	if param <= 16 {
		return 16
	}
	n := param - 1
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	if n < 0 {
		return math.MaxInt32
	}
	return n + 1
}

// 初始化ConcurrentDict
func MakeConcurrent(param int) *ConcurrentDict {
	shardCount := computeCapacity(param)
	table := make([]*shard, shardCount)
	for i := 0; i < shardCount; i++ {
		table[i] = &shard{m: make(map[string]interface{})}
	}
	cd := &ConcurrentDict{
		table:      table,
		count:      0,
		shardCount: shardCount,
	}
	return cd
}

// 设置哈希函数，哈希算法选择

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	for i := 0; i < len(key); i++ {
		hash *= Prime32
		hash ^= uint32(key[i])
	}
	return hash
}

// 定位到某个shard中
func (dict *ConcurrentDict) spread(hashCode uint32) uint32 {
	if dict == nil {
		panic("dict is nil")
	}
	tableSize := uint32(len(dict.table))
	return (tableSize - 1) & hashCode
}

// 获得对应的shard，也就是拿到相应的map
func (dict *ConcurrentDict) getShard(index uint32) *shard {
	if dict == nil {
		panic("dict is nil")
	}
	return dict.table[index]
}

// 下面实现Dict的接口

func (dict *ConcurrentDict) Get(key string) (val interface{}, exists bool) {
	if dict == nil {
		panic("dict is nil")
	}
	// 先计算该key的哈希值
	hashCode := fnv32(key)
	// 定位到某个shard
	index := dict.spread(hashCode)
	// 找到相应的shard
	sh := dict.getShard(index)
	// 对map进行读取的时候加读锁
	sh.mutex.RLock()
	// 解锁
	defer sh.mutex.RUnlock()
	// 获取数据
	val, exists = sh.m[key]
	return
}

func (dict *ConcurrentDict) Len() int {
	if dict == nil {
		panic("dict is nil")
	}
	return int(atomic.LoadInt32(&dict.count))
}

// 往dict中加入数据
func (dict *ConcurrentDict) Put(key string, val interface{}) (result int) {
	if dict == nil {
		panic("dict is nil")
	}
	// 先获得key的哈希值
	hashCode := fnv32(key)
	// 获取shard的index
	index := dict.spread(hashCode)
	// 获得对应的shard
	sh := dict.getShard(index)
	// 加锁,写锁
	sh.mutex.Lock()
	// 解锁
	defer sh.mutex.Unlock()
	// 判断是否存在key值
	if _, ok := sh.m[key]; ok {
		sh.m[key] = val
		return 0
	}
	sh.m[key] = val
	// 不存在的话，先让dict中的数据++
	atomic.AddInt32(&dict.count, 1)
	return 1
}

// 不存在的时候添加啊
func (dict *ConcurrentDict) PutIfAbsent(key string, val interface{}) (result int) {
	if dict == nil {
		panic("dict is nil")
	}
	hashcode := fnv32(key)
	index := dict.spread(hashcode)
	sh := dict.getShard(index)
	sh.mutex.Lock()
	defer sh.mutex.Unlock()
	if _, ok := sh.m[key]; ok {
		return 0
	}
	sh.m[key] = val
	atomic.AddInt32(&dict.count, 1)
	return 1
}

// 存在的时候修改，存在则修改并返回1
func (dict *ConcurrentDict) PutIfExists(key string, val interface{}) (result int) {
	if dict == nil {
		panic("dict is nil")
	}
	hashCode := fnv32(key)
	index := dict.spread(hashCode)
	sh := dict.getShard(index)
	sh.mutex.Lock()
	defer sh.mutex.Unlock()
	if _, ok := sh.m[key]; ok {
		sh.m[key] = val
		return 1
	}
	return 0
}

// 删除节点
func (dict *ConcurrentDict) Remove(key string) (result int) {
	if dict == nil {
		panic("dict is nil")
	}
	hashCode := fnv32(key)
	index := dict.spread(hashCode)
	sh := dict.getShard(index)
	sh.mutex.Lock()
	defer sh.mutex.Unlock()
	if _, ok := sh.m[key]; ok {
		delete(sh.m, key)
		atomic.AddInt32(&dict.count, -1)
		return 1
	}
	return 0
}

// 遍历节点
func (dict *ConcurrentDict) ForEach(consumer Consumer) {
	if dict == nil {
		panic("dict is nil")
	}
	for _, s := range dict.table {
		s.mutex.RLock()
		for key, value := range s.m {
			b := consumer(key, value)
			if !b {
				break
			}
		}
		s.mutex.RUnlock()
	}
}

// 返回所有的key

func (dict *ConcurrentDict) Keys() []string {
	if dict == nil {
		panic("dict is nil")
	}
	i := 0
	keys := make([]string, dict.Len())
	dict.ForEach(func(key string, val interface{}) bool {
		//keys[i] = key
		//i++
		// 采用下面的代码的原因主要是为了应对并发问题
		// 在初始化keys数组后，可能会有新的key加入到dict中
		if i < len(keys) {
			keys[i] = key
			i++
		} else {
			keys = append(keys, key)
		}
		return true
	})
	return keys
}

// 第二种写法
//func (dict *ConcurrentDict) Keys() []string {
//	if dict == nil {
//		panic("dict is nil")
//	}
//	i := 0
//	keys := make([]string, dict.Len())
//	for _, s := range dict.table {
//		s.mutex.RLock()
//		for key := range s.m {
//			keys[i] = key
//			i++
//		}
//		s.mutex.RUnlock()
//	}
//	return keys
//}

// 设置一个函数，随机从shard里面去一个key出来

func (s *shard) RamdomKeyFromShard() string {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	for key := range s.m {
		return key
	}
	return ""
}

// 随机从dict中取limit个keys，可能会包含重复值
// 随机数的取法上做了变化，允许选择相同的shard
// 采用

func (dict *ConcurrentDict) RandomKeys(limit int) []string {
	if dict == nil {
		panic("dict is nil")
	}
	// 结果
	result := make([]string, limit)
	// 一共含有多少个shard
	shardCount := dict.shardCount
	// 添加随机种子
	nR := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < limit; {
		// 获取shard,可能获取到的shard是没有任何元素的，下面需要进一步判断
		sh := dict.getShard(uint32(nR.Intn(shardCount)))
		// 如果shard没有初始化
		if sh == nil {
			continue
		}
		key := sh.RamdomKeyFromShard()
		// 选到的shard里面可能是什么都没有存储有的，所以key可能为”“
		// 如果key为空则不进行i++，继续选择随机数，然后继续随机生成shard，进行取值
		if key != "" {
			result[i] = key
			i++
		}
	}
	return result
}

// 随机获取limit个key值，保证key不能重复
func (dict *ConcurrentDict) RandomDistinctKeys(limit int) []string {
	if dict == nil {
		panic("dict is nil")
	}
	if limit >= dict.Len() {
		return dict.Keys()
	}
	result := make([]string, limit)
	// 定义一个map，用来存储已经拿出来的key
	existKeyMap := make(map[string]struct{}, limit)
	// 产生随机不重复的数字
	nR := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < limit; {
		sh := dict.getShard(uint32(nR.Intn(dict.shardCount)))
		// 如果sh==nil跳过
		if sh == nil {
			continue
		}
		// 随机生成一个key
		key := sh.RamdomKeyFromShard()
		// 判断key的值
		if key != "" {
			if _, exist := existKeyMap[key]; !exist { // 如果当前的key没有在map中则添加，说明没有遍历过该key
				existKeyMap[key] = struct{}{}
				result[i] = key
				i++
			}
		}
	}
	return result
}

// 清空dict，则直接新建一个dict即可，旧的dict让GC回收

func (dict *ConcurrentDict) Clear() {
	*dict = *MakeConcurrent(dict.shardCount)
}
