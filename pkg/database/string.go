package database

import (
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"math/bits"
	"slava/internal/interface/database"
	"slava/internal/interface/slava"
	"slava/internal/protocol"
	"slava/internal/utils"
	"slava/pkg/datastruct/bitmap"

	. "slava/internal/constData"
)

// 需要实现的功能如下：
// Get 获取某个键的值
// GetEX 获取某个键的值并且设置该键值的过期时间
// Set 设置键和值，以及设置该key-value的过期时间
// SetNX 如果不存在则插入key value并且返回1，如果存在则返回0，
// SetEX 添加键值并且附带过期时间（以秒位单位）
// PSetEX 添加键值并且以毫秒的位单位添加过期时间
// MSet 一次设置多个k-v
// MGet 一次获取多个k-v
// MSetNX 只有当所有的key不存在的时候，设置多个k-v。
// GetSet 以旧换新，设置一个key的新值，并且返回该ky的旧值
// GetDel 先获取key的value然后再删除该键
// Incr key所对应的value加一，如果不是value不是整数则失败
// IncrBy key所对应的value加上给定的值，如果不是整数则失败
// IncrByFloat key所对应的value增加一个给定的浮点值
// Decr key所对应的value，减1
// Decr key所对应的value减少给定的值
// StrLen 返回key对应的value的长度
// Append key对应的value追加字符串
// SetRange 覆盖存储在key处的字符串的一部分，从指定的偏移量开始。如果偏移量大于键处字符串的当前长度，则字符串将填充零字节。
// GetRange 获得key对应value的范围的值
// Setbit
// GetBit
// BitCount
// BitPos
//

func (db *DB) getAsString(key string) ([]byte, protocol.ErrorReply) {
	entity, exists := db.GetEntity(key)
	if !exists {
		return nil, nil
	}
	bytes, ok := entity.Data.([]byte)
	if !ok {
		return nil, &protocol.WrongTypeErrReply{}
	}
	return bytes, nil
}

// Get 获取某个键的值
func execGet(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	bytes, reply := db.getAsString(key)
	if reply != nil {
		return reply
	}
	if bytes == nil { // 表示没获取到key所对应的value，可能因为key过期了

		return protocol.MakeNullBulkReply()
	}
	return protocol.MakeBulkReply(bytes)
}

// 设置一些常数
//const unlimitedTTl int64 = 0
//const (
//	upsertPolicy = iota // default
//	insertPolicy        // set nx
//	updatePolicy        // set ex
//)

// GetEX 获取某个键的值并且设置该键值的过期时间
func execGetEX(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	bytes, err := db.getAsString(key)
	ttl := UnlimitedTTl
	if err != nil {
		return err
	}
	if bytes == nil { // 说明键已经过期，没获取到key的值
		return protocol.MakeNullBulkReply()
	}
	for i := 1; i < len(args); i++ {
		arg := strings.ToUpper(string(args[1]))
		if arg == "EX" { // second 单位是秒
			if ttl != UnlimitedTTl { // 说明该键值已经设有过期时间了
				return &protocol.SyntaxErrReply{}
			}
			if i+1 >= len(args) { // 参数不够
				return &protocol.SyntaxErrReply{}
			}
			ttlArg, err := strconv.ParseInt(string(args[i+1]), 10, 64)
			if err != nil {
				return &protocol.SyntaxErrReply{}
			}
			if ttlArg <= 0 {
				return protocol.MakeErrReply("ERR invalid expire time in getnx")
			}
			ttl = ttlArg * 1000
		} else if arg == "PX" { // 单位是
			if ttl != UnlimitedTTl {
				return &protocol.SyntaxErrReply{}
			}
			if i+1 >= len(args) {
				return &protocol.SyntaxErrReply{}
			}
			ttlArg, err := strconv.ParseInt(string(args[i+1]), 10, 64)
			if err != nil {
				return &protocol.SyntaxErrReply{}
			}
			if ttlArg <= 0 {
				return protocol.MakeErrReply("ERR invalid expire time in getnx")
			}
			ttl = ttlArg
		} else if arg == "PERSIST" { // 取消某个键的ttlCmd，不能和EX和PX共用。Redis PERSIST 命令用于移除给定 key 的过期时间，使得 key 永不过期。
			if ttl != UnlimitedTTl { // PERSIST和EX或者PX不能一块使用
				return &protocol.SyntaxErrReply{}
			}
			if i+1 != len(args) { // getnx key persist
				return &protocol.SyntaxErrReply{}
			}
			db.Persist(key)
		} else if arg == "EXAT" {
			if ttl != UnlimitedTTl { // 说明该键值已经设有过期时间了
				return &protocol.SyntaxErrReply{}
			}
			if i+1 >= len(args) { // 参数不够
				return &protocol.SyntaxErrReply{}
			}
			ttlArg, err := strconv.ParseInt(string(args[i+1]), 10, 64)
			if err != nil {
				return &protocol.SyntaxErrReply{}
			}
			//if ttlArg <= 0 {
			//	return protocol.MakeErrReply("ERR invalid expire time in getnx")
			//}
			nowTime := time.Now().Unix()
			if ttlArg-nowTime <= 0 {
				return protocol.MakeErrReply("ERR invalid expire time in getnx")
			}
			ttl = (ttlArg - nowTime) * 1000
		} else if arg == "PXAT" {
			if ttl != UnlimitedTTl { // 说明该键值已经设有过期时间了
				return &protocol.SyntaxErrReply{}
			}
			if i+1 >= len(args) { // 参数不够
				return &protocol.SyntaxErrReply{}
			}
			ttlArg, err := strconv.ParseInt(string(args[i+1]), 10, 64)
			if err != nil {
				return &protocol.SyntaxErrReply{}
			}
			// 变成毫秒
			nowTime := time.Now().Unix() * 1000
			if (ttlArg - nowTime) <= 0 {
				return protocol.MakeErrReply("ERR invalid expire time in getnx")
			}
			ttl = ttlArg - nowTime
		} else {
			return &protocol.SyntaxErrReply{}
		}
	}
	if len(args) > 1 { // 说明key后面还存在参数
		if ttl != UnlimitedTTl { // 说明key参数后面跟了EX或PX
			expireTime := time.Now().Add(time.Duration(ttl) * time.Millisecond) // Add函数传入的值是纳秒
			db.Expire(key, expireTime)
			// 加入AOF操作
			// TODO
		} else { // len(args) > 1 并且ttl == unlimitedTTl说明此时key后面的参数一定是PERSIST，但是在上面已经执行过PERSIST的操作了
			// 加入AOF操作
			db.addAof(utils.ToCmdLine3("persist", args[0]))
		}
	}
	return protocol.MakeBulkReply(bytes)
}

// Set 设置键和值，以及设置该key-value的过期时间
// set key value；set key value [ex seconds][PX milliseconds][NX|XX]
// EX second ：设置键的过期时间为 second 秒。 SET key value EX second 效果等同于 SETEX key second value
// PX millisecond ：设置键的过期时间为 millisecond 毫秒。 SET key value PX millisecond 效果等同于 PSETEX key millisecond value
// NX ：只在键不存在时，才对键进行设置操作。 SET key value NX 效果等同于 SETNX key value
// XX ：只在键已经存在时，才对键进行设置操作。

// 后面跟的EX、PX、NX、XX只能是其中一个

// 对于某个原本带有生存时间（TTL）的键来说， 当 SET 命令成功在这个键上执行时， 这个键原有的 TTL 将被清除。

func execSet(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	value := args[1]
	policy := UpsertPolicy
	ttl := UnlimitedTTl
	// 如果value后面还存在参数，解析value后面的参数
	if len(args) > 2 {
		for i := 2; i < len(args); i++ {
			arg := strings.ToUpper(string(args[i]))
			if arg == "NX" { // 表示如果不存在则插入
				if policy == UpdatePolicy { // 如果当前的是处于XX的状态，但是又遇到NX则报错
					return &protocol.SyntaxErrReply{}
				}
				policy = InsertPolicy
			} else if arg == "XX" {
				if policy == InsertPolicy { // 如果当前是NX状态,但是又遇到XX则报错
					return &protocol.SyntaxErrReply{}
				}
				policy = UpdatePolicy
			} else if arg == "EX" {
				if ttl != UnlimitedTTl { // 表明再遍历中已经遇到过一个EX或者PX命令，现在又遇到一个，所以报错
					return &protocol.SyntaxErrReply{}
				}
				if i+1 >= len(arg) {
					return &protocol.SyntaxErrReply{}
				}
				tllArg, err := strconv.ParseInt(string(args[i+1]), 10, 64)
				if err != nil {
					return &protocol.SyntaxErrReply{}
				}
				ttl = tllArg * 1000
				i++ // 跳过失效的时间
			} else if arg == "PX" {
				if ttl != UnlimitedTTl {
					return &protocol.SyntaxErrReply{}
				}
				if i+1 >= len(arg) {
					return &protocol.SyntaxErrReply{}
				}
				ttlArg, err := strconv.ParseInt(string(args[i+1]), 10, 64)
				if err != nil {
					return &protocol.SyntaxErrReply{}
				}
				ttl = ttlArg
				i++
			} else { // 如果value后面的参数既不是NX、XX、EX、PX，则报错
				return &protocol.SyntaxErrReply{}
			}
		}
	}
	entity := &database.DataEntity{Data: value}
	result := 0
	switch policy {
	case UpsertPolicy: // 默认值，说明后面跟的不是EX和XX
		db.PutEntity(key, entity) // 执行了普通的set，可能后面会跟EX或PX，要进行判断一下
		// 就算上面db.PutEntity(key, entity)返回的结果是0，也要对后面对EX和PX的判断
		if ttl != UnlimitedTTl { // 说明后面跟了EX或者PX
			expireTime := time.Now().Add(time.Duration(ttl) * time.Millisecond)
			db.Expire(key, expireTime)
			// AOF
			// TODO
		}
	case InsertPolicy: // NX,如果不存在则插入
		result = db.PutIfAbsent(key, entity)

		if result == 1 { // 表明已经进行了操作
			// AOF
			db.addAof(utils.ToCmdLine3("set", args...))
		}
	case UpdatePolicy: // XX,存在则操作
		result = db.PutIfExists(key, entity)
		if result == 1 { // 表明对现有的key进行了修改
			db.Persist(key) // 对于某个原本带有生存时间（TTL）的键来说， 当 SET 命令成功在这个键上执行时， 这个键原有的 TTL 将被清除。
			// 在Persist中会判断key是不是在ttMap中
			// AOF操作
			db.addAof(utils.ToCmdLine3("set", args...))
		}
		//db.Persist(key) // 由于Persist里面会判断key存不存在于ttlMap中，所以这里不用在判断
	}

	//if result > 0 {
	//	if ttl != unlimitedTTl {
	//		expireTime := time.Now().Add(time.Duration(ttl) * time.Millisecond)
	//		db.Expire(key, expireTime)
	//		db.addAof(Cmdline{
	//			[]byte("SET"),
	//			args[0],
	//			args[1],
	//		})
	//		db.addAof(aof.MakeExpireCmd(key, expireTime).Args)
	//	} else {
	//		db.Persist(key) // override ttl
	//		db.addAof(utils.ToCmdLine3("set", args...))
	//	}
	//}

	if result > 0 {
		return &protocol.OkReply{}
	}
	return &protocol.NullBulkReply{}
}

// SetNX 如果不存在则插入key value并且返回1，如果存在则返回0，

func execSetNX(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	value := args[1]
	entity := &database.DataEntity{Data: value}
	result := db.PutIfAbsent(key, entity)
	db.addAof(utils.ToCmdLine3("setnx", args...))
	return protocol.MakeIntReply(int64(result))
}

// SetEX 添加键值并且附带过期时间（以秒位单位）
func execSetEX(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	value := args[2]
	ttlArg, err := strconv.ParseInt(string(args[1]), 10, 64)
	if err != nil {
		return &protocol.SyntaxErrReply{}
	}
	if ttlArg <= 0 {
		return protocol.MakeErrReply("ERR invalid expire time in setex")
	}
	ttl := ttlArg * 1000
	entity := &database.DataEntity{Data: value}
	db.PutEntity(key, entity)
	expireTime := time.Now().Add(time.Duration(ttl) * time.Millisecond)
	db.Expire(key, expireTime)
	db.addAof(utils.ToCmdLine3("setex", args...))
	// AOF
	// TODO
	return &protocol.OkReply{}
}

// PSetEX 添加键值并且以毫秒的位单位添加过期时间
func execPSetEX(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	value := args[2]
	ttlArg, err := strconv.ParseInt(string(args[1]), 10, 64)
	if err != nil {
		return &protocol.SyntaxErrReply{}
	}
	if ttlArg <= 0 {
		return protocol.MakeErrReply("ERR invalid expire time in psetex")
	}
	entity := &database.DataEntity{Data: value}
	db.PutEntity(key, entity)
	expireTime := time.Now().Add(time.Duration(ttlArg) * time.Millisecond)
	db.Expire(key, expireTime)
	db.addAof(utils.ToCmdLine3("setex", args...))
	// AOF
	// TODO
	return &protocol.OkReply{}
}

// MSet 一次设置多个k-v

func prepareMSet(args [][]byte) ([]string, []string) {
	size := len(args) / 2
	keys := make([]string, size)
	for i := 0; i < size; i++ {
		keys[i] = string(args[2*i])
	}
	return keys, nil
}

func undoMSet(db *DB, args [][]byte) []Cmdline {
	weiteKeys, _ := prepareMSet(args)
	return rollbackGivenKeys(db, weiteKeys...)
}

func execMSet(db *DB, args [][]byte) slava.Reply {
	if len(args)%2 != 0 {
		return protocol.MakeSyntaxErrReply()
	}
	size := len(args) / 2
	keys := make([]string, size)
	values := make([][]byte, size)
	for i := 0; i < size; i++ {
		keys[i] = string(args[2*i])
		values[i] = args[2*i+1]
	}
	for i := 0; i < size; i++ {
		entity := &database.DataEntity{
			Data: values[i],
		}
		db.PutEntity(keys[i], entity)
	}
	db.addAof(utils.ToCmdLine3("mset", args...))
	return &protocol.OkReply{}
}

func execMGet(db *DB, args [][]byte) slava.Reply {
	keys := make([]string, len(args))
	result := make([][]byte, len(args))
	for i := 0; i < len(args); i++ {
		keys[i] = string(args[i])
		bytes, err := db.getAsString(keys[i])
		if err != nil {
			_, isWrongType := err.(*protocol.WrongTypeErrReply)
			if isWrongType { // 类型断言成功
				result[i] = nil
				continue
			} else { // 类型断言没成功
				return err
			}
		}
		// 如果key已经过期则bytes等于nil，不管是不是nil都会加入到result中
		result[i] = bytes
	}
	return protocol.MakeMultiBulkReply(result)
}

// MSetNX 只有当所有的key不存在的时候，设置多个k-v。
func execMSetNX(db *DB, args [][]byte) slava.Reply {
	if len(args)%2 != 0 {
		return protocol.MakeSyntaxErrReply()
	}
	size := len(args) / 2

	keys := make([]string, size)
	values := make([][]byte, size)
	for i := 0; i < size; i++ {
		keys[i] = string(args[2*i])
		values[i] = args[2*i+1]
	}
	for _, key := range keys {
		_, ok := db.GetEntity(key)
		if ok {
			return protocol.MakeIntReply(0)
		}
	}
	for i, key := range keys {
		entity := &database.DataEntity{Data: values[i]}
		db.PutEntity(key, entity)
	}
	db.addAof(utils.ToCmdLine3("msetnx", args...))
	return protocol.MakeIntReply(1)
}

// GetSet 以旧换新，设置一个key的新值，并且返回该ky的旧值

func execGetSet(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	value := args[1]
	// 给定键值然后获取相应的value，返回的值是一个【】byte，和一个err
	entity, err := db.getAsString(key)
	if err != nil {
		return err
	}
	db.PutEntity(key, &database.DataEntity{Data: value})
	// 修改了key的值，这时候就需要重置key的ttl
	db.Persist(key)
	db.addAof(utils.ToCmdLine3("getset", args...))
	if entity == nil { // 说明其中没有key的值
		return &protocol.NullBulkReply{}
	}
	return protocol.MakeBulkReply(entity)
}

// GetDel 先获取key的value然后再删除该键
func execGetDel(db *DB, args [][]byte) slava.Reply {
	keys := string(args[0])
	value, err := db.getAsString(keys)
	if err != nil { // key存在，但是如果key的类型不是string则返回err
		return err
	}
	if value == nil { // 表示key不存在
		return &protocol.NullBulkReply{}
	}
	// 删除key值
	db.Remove(keys)
	db.addAof(utils.ToCmdLine3("getdel", args...))
	return protocol.MakeBulkReply(value)
}

// Incr key所对应的value加一，如果不是value不是整数则失败
func execIncr(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	value, err := db.getAsString(key)
	if err != nil { // key值存在但是value不是string类型
		return err
	}
	if value == nil { // key值不存在，没有找到key的值，先加入key的值key的值最开始是0，然后incr
		db.PutEntity(key, &database.DataEntity{Data: []byte("1")})
		db.addAof(utils.ToCmdLine3("incr", args...))
		return protocol.MakeIntReply(1)
	}
	Num, err2 := strconv.ParseInt(string(value), 10, 64)
	if err2 != nil {
		return protocol.MakeErrReply("ERR value is not an integer or out of range")
	}
	newValue := strconv.FormatInt(Num+1, 10)
	db.PutEntity(key, &database.DataEntity{Data: []byte(newValue)})
	db.addAof(utils.ToCmdLine3("incr", args...))
	return protocol.MakeIntReply(Num + 1)
}

// IncrBy key所对应的value加上给定的值，如果不是整数则失败
func execIncrBy(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	rawDelta := string(args[1])
	delta, err := strconv.ParseInt(rawDelta, 10, 64)
	if err != nil {
		return protocol.MakeErrReply("ERR value is not an integer or out of range")
	}
	// 取key中对应的value
	value, errorReply := db.getAsString(key)
	if errorReply != nil { // key值存在但是value不是string类型
		return errorReply
	}
	if value == nil { // key值不存在，没有找到key的值，先加入key的值key的值最开始是0，然后incr
		db.PutEntity(key, &database.DataEntity{Data: args[1]})
		db.addAof(utils.ToCmdLine3("incrby", args...))
		return protocol.MakeIntReply(delta)
	}
	Num, err2 := strconv.ParseInt(string(value), 10, 64)
	if err2 != nil {
		return protocol.MakeErrReply("ERR value is not an integer or out of range")
	}
	newValue := strconv.FormatInt(Num+delta, 10)
	db.PutEntity(key, &database.DataEntity{Data: []byte(newValue)})
	db.addAof(utils.ToCmdLine3("incrby", args...))
	return protocol.MakeIntReply(Num + delta)
}

// IncrByFloat key所对应的value增加一个给定的浮点值
func execIncrByFloat(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	rawDelta := string(args[1])
	delta, err := decimal.NewFromString(rawDelta)
	if err != nil {
		return protocol.MakeErrReply("ERR value is not a valid float")
	}
	value, errReply := db.getAsString(key)
	if errReply != nil { // key的value不是string
		return errReply
	}
	if value == nil { // 说明key不存在，则要添加key
		db.PutEntity(key, &database.DataEntity{Data: args[1]})
		db.addAof(utils.ToCmdLine3("incrfloat", args...))
		return protocol.MakeBulkReply(args[1])
	}
	val, err := decimal.NewFromString(string(value))
	if err != nil {
		return protocol.MakeErrReply("ERR value is not a valid float")
	}
	db.PutEntity(key, &database.DataEntity{Data: []byte(delta.Add(val).String())})
	db.addAof(utils.ToCmdLine3("incrfloat", args...))
	return protocol.MakeBulkReply([]byte(delta.Add(val).String()))
}

// Decr key所对应的value，减1
func execDecr(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	value, errorReply := db.getAsString(key)
	if errorReply != nil { // key值存在但是value的值不是string类型
		return errorReply
	}
	if value == nil { // 表明key不存在
		db.PutEntity(key, &database.DataEntity{Data: []byte("-1")})
		db.addAof(utils.ToCmdLine3("decr", args...))
		return protocol.MakeIntReply(-1)
	}
	val, err := strconv.ParseInt(string(value), 10, 64)
	if err != nil {
		return protocol.MakeErrReply("ERR value is not an integer or out of range")
	}
	db.PutEntity(key, &database.DataEntity{Data: []byte(strconv.FormatInt(val-1, 10))})
	db.addAof(utils.ToCmdLine3("decr", args...))
	return protocol.MakeIntReply(val - 1)
}

// DecrBy key所对应的value减少给定的值
func execDecrBy(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	rawDelta := string(args[1])
	delta, err := strconv.ParseInt(rawDelta, 10, 64)
	if err != nil {
		return protocol.MakeErrReply("ERR value is not an integer or out of range")
	}
	value, errReply := db.getAsString(key)
	if errReply != nil { // 找到key但是value不是string
		return errReply
	}
	val, err := strconv.ParseInt(string(value), 10, 64)
	if err != nil {
		return protocol.MakeErrReply("ERR value is not an integer or out of range")
	}
	if value == nil { // 没有找到key
		db.PutEntity(key, &database.DataEntity{Data: []byte(strconv.FormatInt(-delta, 10))})
		db.addAof(utils.ToCmdLine3("decrBy", args...))
		return protocol.MakeIntReply(-delta)
	}
	db.PutEntity(key, &database.DataEntity{Data: []byte(strconv.FormatInt(val-delta, 10))})
	db.addAof(utils.ToCmdLine3("decrBy", args...))
	return protocol.MakeIntReply(val - delta)
}

// StrLen 返回key对应的value的长度
func execStrLen(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	value, errorReply := db.getAsString(key)
	if errorReply != nil { // value不是字符串
		return errorReply
	}
	if value == nil { // key不存在
		return protocol.MakeIntReply(0)
	}
	return protocol.MakeIntReply(int64(len(value)))
}

// Append key对应的value追加字符串
func execAppend(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	appendValue := args[1]
	bytes, err := db.getAsString(key)
	if err != nil {
		return err
	}
	// 可以不需要下面的bytes == nil的判断
	//if bytes == nil { // 不存在key
	//	db.PutEntity(key, &database.DataEntity{Data: appendValue})
	//	db.addAof(utils.ToCmdLine3("execappend", args...))
	//	return protocol.MakeIntReply(int64(len(appendValue)))
	//}

	// 不管key存在与否都进行下面的操作
	bytes = append(bytes, appendValue...)
	db.PutEntity(key, &database.DataEntity{Data: bytes})
	db.addAof(utils.ToCmdLine3("append", args...))
	return protocol.MakeIntReply(int64(len(bytes)))
}

// SetRange 覆盖存储在key处的字符串的一部分，从指定的偏移量开始。如果偏移量大于键处字符串的当前长度，则字符串将填充零字节，填充完后再append。

func execSetRange(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	indexStr := string(args[1])
	// 要添加的value
	valueRange := args[2]
	index, err := strconv.ParseInt(indexStr, 10, 64)
	if err != nil {
		return protocol.MakeErrReply(err.Error())
	}
	value, errorReply := db.getAsString(key)
	if errorReply != nil {
		return errorReply
	}
	// 即使value是nil也可以对他进行求len
	if index > int64(len(value)) {
		diff := index - int64(len(value))
		diffArray := make([]byte, diff)
		value = append(value, diffArray...)
		db.PutEntity(key, &database.DataEntity{Data: value})
	}
	valueLen := len(value)
	for i := 0; i < len(valueRange); i++ {
		if index < int64(valueLen) {
			value[index] = valueRange[i]
			index++
		} else {
			value = append(value, valueRange[i])
		}
	}
	db.PutEntity(key, &database.DataEntity{Data: value})
	db.addAof(utils.ToCmdLine3("setRange", args...))
	return protocol.MakeIntReply(int64(len(value)))
}

// GetRange 获得key对应value的范围的值
func execGetRange(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	startIndexStr := string(args[1])
	endIndexStr := string(args[2])
	startIndex, errStartIndex := strconv.ParseInt(startIndexStr, 10, 64)
	if errStartIndex != nil {
		return protocol.MakeErrReply("ERR value is not an integer or out of range")
	}
	endIndex, errEndIndex := strconv.ParseInt(endIndexStr, 10, 64)
	if errEndIndex != nil {
		return protocol.MakeErrReply("ERR value is not an integer or out of range")
	}
	// key不存在，不管start和end谁大谁小始终会返回空字符串
	bytes, err := db.getAsString(key)
	if err != nil {
		return err
	}
	if bytes == nil {
		return protocol.MakeNullBulkReply()
	}
	bytesLen := len(bytes)
	// utils包下面的ConvertRange函数可以检验startIndex和endIndex是否合法
	begin, end := utils.ConvertRange(startIndex, endIndex, int64(bytesLen))
	if begin == -1 {
		return protocol.MakeNullBulkReply()
	}
	return protocol.MakeBulkReply(bytes[begin:end])

}

// Setbit

func execSetBit(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	offset, err := strconv.ParseInt(string(args[1]), 10, 64)
	if err != nil {
		return protocol.MakeErrReply("ERR bit offset is not an integer or out of range")
	}
	valStr := string(args[2])
	var v byte
	if valStr == "1" {
		v = 1
	} else if valStr != "0" {
		return protocol.MakeErrReply("ERR bit is not an integer or out of range")
	}
	bytes, errorReply := db.getAsString(key)
	if errorReply != nil {
		return errorReply
	}
	bm := bitmap.FromBytes(bytes)
	former := bm.GetBit(offset)
	bm.SetBit(offset, v)
	db.PutEntity(key, &database.DataEntity{Data: bm.ToBytes()})
	return protocol.MakeIntReply(int64(former))
}

// GetBit
func execGetBit(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	offset, err := strconv.ParseInt(string(args[1]), 10, 64)
	if err != nil {
		return protocol.MakeErrReply("ERR bit offset is not an integer or out of range")
	}
	bytes, errorReply := db.getAsString(key)
	if errorReply != nil {
		return errorReply
	}
	bm := bitmap.FromBytes(bytes)
	return protocol.MakeIntReply(int64(bm.GetBit(offset)))
}

// BitCount
func execBitCount(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	bytes, errorReply := db.getAsString(key)
	if errorReply != nil {
		return errorReply
	}
	// 不存在的 key 被当成是空字符串来处理，因此对一个不存在的 key 进行 BITCOUNT 操作，结果为 0 。
	if bytes == nil {
		return protocol.MakeIntReply(0)
	}
	byteMode := true
	if len(args) > 3 {
		mode := strings.ToUpper(string(args[3]))
		if mode == "BIT" {
			byteMode = false
		} else if mode == "BYTE" {
			byteMode = true
		} else {
			return protocol.MakeErrReply("ERR syntax error")
		}
	}
	var size int64
	bm := bitmap.FromBytes(bytes)
	if byteMode {
		size = int64(len(*bm))
	} else {
		size = int64(bm.BitSize())
	}
	var begin int
	var end int
	if len(args) > 1 {
		startIndex, err1 := strconv.ParseInt(string(args[1]), 10, 64)
		if err1 != nil {
			return protocol.MakeErrReply("ERR value is not an integer or out of range")
		}
		endeIndex, err2 := strconv.ParseInt(string(args[2]), 10, 64)
		if err2 != nil {
			return protocol.MakeErrReply("ERR value is not an integer or out of range")
		}
		begin, end = utils.ConvertRange(startIndex, endeIndex, size)
		if begin == -1 {
			return protocol.MakeIntReply(0)
		}
	}
	var count int64
	if byteMode {
		bm.ForEachByte(begin, end, func(offset int64, val byte) bool {
			count += int64(bits.OnesCount8(val))
			return true
		})
	} else {
		bm.ForEachBit(int64(begin), int64(end), func(offset int64, val byte) bool {
			if val > 0 {
				count++
			}
			return true
		})
	}
	return protocol.MakeIntReply(count)
}

// BitPos

func execBitPos(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])                   // key值
	bytes, errorReply := db.getAsString(key) // 取value
	if errorReply != nil {
		return errorReply
	}
	if bytes == nil { // value 不存在返回-1
		return protocol.MakeIntReply(-1)
	}
	valStr := string(args[1])

	var v byte // 判断取1还是0

	if valStr == "1" {
		v = 1
	} else if valStr == "0" {
		v = 0
	} else {
		return protocol.MakeErrReply("ERR bit is not an integer or out of range")
	}
	// 字节模式还是比特模式，默认是字节模式
	byteMode := true
	if len(args) > 4 {
		mode := strings.ToLower(string(args[4]))
		if mode == "bit" {
			byteMode = false
		} else if mode == "byte" {
			byteMode = true
		} else {
			return protocol.MakeErrReply("ERR syntax error")
		}
	}
	// 计算字节数或者bit数，根据所给的模式
	var size int64
	bm := bitmap.FromBytes(bytes)
	if byteMode {
		size = int64(len(*bm))
	} else {
		size = int64(bm.BitSize())
	}
	// 查找位置，遍历的范围
	var begin, end int
	if len(args) > 2 {
		startIndex, err1 := strconv.ParseInt(string(args[2]), 10, 64)
		if err1 != nil {
			return protocol.MakeErrReply("ERR value is not an integer or out of range")
		}
		endIndex, err2 := strconv.ParseInt(string(args[3]), 10, 64)
		if err2 != nil {
			return protocol.MakeErrReply("ERR value is not an integer or out of range")
		}
		begin, end = utils.ConvertRange(startIndex, endIndex, size)
		// 遍历范围不合法
		if begin == -1 {
			return protocol.MakeIntReply(-1)
		}
	}
	if byteMode {
		begin *= 8
		end *= 8
	}
	var offset = int64(-1)
	bm.ForEachBit(int64(begin), int64(end), func(o int64, val byte) bool {
		if val == v { // 如果找到第一个v的值，然后就可以返回了
			offset = o
			return false
		}
		return true
	})
	return protocol.MakeIntReply(offset)
}

// init的方法，将上面的方法注册到cmdTable中
func init() {
	// set k v ex time|px time|xx|nx
	RegisterCommand("Set", execSet, writeFirstKey, rollbackFirstKey, ArityNegativeTree, FlagWrite)
	// setnx k v
	RegisterCommand("SetNX", execSetNX, writeFirstKey, rollbackFirstKey, ArityTree, FlagWrite)
	// setex k time v
	RegisterCommand("SetEX", execSetEX, writeFirstKey, rollbackFirstKey, ArityFour, FlagWrite)
	// psetex k time v
	RegisterCommand("PSetEX", execPSetEX, writeFirstKey, rollbackFirstKey, ArityFour, FlagWrite)
	// mset k1 v1 k2 v2 k3 v3 ...
	RegisterCommand("MSet", execMSet, prepareMSet, undoMSet, ArityNegativeTree, FlagWrite)
	// msetnx k1 v1 k2 v2 k3 v3 ...
	RegisterCommand("MSetNX", execMSetNX, prepareMSet, undoMSet, ArityNegativeTree, FlagWrite)
	//MGet k1 k2 k3 ...
	RegisterCommand("MGet", execMGet, readAllKeys, nil, ArityNegativeTwo, FlagReadOnly)
	// get k
	RegisterCommand("Get", execGet, readFirstKey, nil, ArityTwo, FlagReadOnly)
	// GETEX key [EX seconds | PX milliseconds | EXAT unix-time-seconds | PXAT unix-time-milliseconds | PERSIST]
	RegisterCommand("GetEX", execGetEX, writeFirstKey, rollbackFirstKey, ArityNegativeTree, FlagWrite)
	// getset key value
	RegisterCommand("GetSet", execGetSet, writeFirstKey, rollbackFirstKey, ArityTree, FlagWrite)
	// getdel key
	RegisterCommand("GetDel", execGetDel, writeFirstKey, rollbackFirstKey, ArityTwo, FlagWrite)
	// incr key
	RegisterCommand("Incr", execIncr, writeFirstKey, rollbackFirstKey, ArityTwo, FlagWrite)
	// incrby key int
	RegisterCommand("IncrBy", execIncrBy, writeFirstKey, rollbackFirstKey, ArityTree, FlagWrite)
	// incrbyfloat ket float
	RegisterCommand("IncrByFloat", execIncrByFloat, writeFirstKey, rollbackFirstKey, ArityTree, FlagWrite)
	// decr key
	RegisterCommand("Decr", execDecr, writeFirstKey, rollbackFirstKey, ArityTwo, FlagWrite)
	// decr key int
	RegisterCommand("DecrBy", execDecrBy, writeFirstKey, rollbackFirstKey, ArityTree, FlagWrite)
	// strlen key
	RegisterCommand("StrLen", execStrLen, readFirstKey, nil, ArityTwo, FlagReadOnly)
	// apend key value
	RegisterCommand("Append", execAppend, writeFirstKey, rollbackFirstKey, ArityTree, FlagWrite)
	// setrange key offset value
	RegisterCommand("SetRange", execSetRange, writeFirstKey, rollbackFirstKey, ArityFour, FlagWrite)
	// getrange key begin end
	RegisterCommand("GetRange", execGetRange, readFirstKey, nil, ArityFour, FlagReadOnly)
	// setbit key offset v
	RegisterCommand("SetBit", execSetBit, writeFirstKey, rollbackFirstKey, ArityFour, FlagWrite)
	// getbit key offset
	RegisterCommand("GetBit", execGetBit, readFirstKey, nil, ArityTree, FlagReadOnly)
	// bitcount key (begin end)
	RegisterCommand("BitCount", execBitCount, readFirstKey, nil, ArityNegativeTwo, FlagReadOnly)
	// bitpos key v begin end
	RegisterCommand("BitPos", execBitPos, readFirstKey, nil, ArityNegativeTree, FlagReadOnly)

}
