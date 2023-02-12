package constdata

// 这个包用来存放项目的常数

// 在concurrent的包下：设置哈希函数，哈希算法选择

const Prime32 = uint32(16777619)

// 在router包下：用来指示exec函数是写还是只读
const (
	FlagWrite = iota
	FlagReadOnly
)

// database包下面的常数：用于初始化DB
const (
	DataDictSize = 1 << 16 // 表示的是分数据库的大小
	TtlDictSize  = 1 << 10 // 过期时间的大小
	LockerSize   = 1024
)

// string包下面的常数
// 表示默认的过期时间

const UnlimitedTTl int64 = 0

// 用于指示是nx还是xx
const (
	UpsertPolicy = iota // default
	InsertPolicy        // set nx
	UpdatePolicy        // set xx
)

// 用于string包下面的RegisterCommand函数arity字段
const (
	ArityTwo          = 2
	ArityTree         = 3
	ArityFour         = 4
	ArityNegativeTwo  = -2
	ArityNegativeTree = -3
	ArityNegativeFour = -4
)
