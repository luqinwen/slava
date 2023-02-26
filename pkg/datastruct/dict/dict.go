package dict

// 定义一个Consumer函数，该函数是用来遍历dict，如果在该函数中返回fasle则停止遍历
type Consumer func(key string, val interface{}) bool

// 定义一个接口，slava底层用的数据结构，map
// 定义一个接口的目的就是，在实现的时候可以根据需要选择不一样数据结构进行实现，比如普通的map、sync.Map等
// 定义接口，有利于改善和迭代项目

type Dict interface {
	Get(key string) (val interface{}, exists bool)        // 获取key对应的value
	Len() int                                             // 数据结构中多少个数据
	Put(key string, val interface{}) (result int)         // 加入k-v，如果原先的key已经存在则会覆盖原来的value，并且返回0，如果key不存在则加入k-v，并且返回1
	PutIfAbsent(key string, val interface{}) (result int) // 如果数据结构中不存在k-v则加入k-v并且返回1，如果存在则返回0
	PutIfExists(key string, val interface{}) (result int) // 如果数据结构中存在k-v，则覆盖掉原来的value，返回1，如果不存在则不操作，返回0
	Remove(key string) (result int)                       // 删除某个k-v，如果存在则删除，并返回1，如果不存在则返回0
	ForEach(consumer Consumer)                            // 遍历整个dict
	Keys() []string                                       // 返回dict中所有的key
	RandomKeys(limit int) []string                        // 返回limit数量的任意key值,可以出现重复的值
	RandomDistinctKeys(limit int) []string                // 返回limit数量的key，要求key不重复
	Clear()                                               // 清空dict中的数据
}
