package database

import (
	"strings"
	"sync/atomic"
	"time"

	. "slava/internal/data"
	"slava/internal/interface/database"
	"slava/internal/interface/slava"
	"slava/internal/protocol"
	"slava/pkg/datastruct/dict"
	"slava/pkg/datastruct/lock"
	"slava/pkg/logger"
	"slava/pkg/timewheel"
)

// 分数据库的功能，slava含有16个分数据库

//const (
//	dataDictSize = 1 << 16 // 表示的是分数据库的大小
//	ttlDictSize  = 1 << 10 // 过期时间的大小
//	lockerSize   = 1024
//)

// TODO DB字段设置
type DB struct {
	index      int       // 表示第几个分数据库
	data       dict.Dict // 分数据库的数据存放
	ttlMap     dict.Dict // key所对应的过期时间
	versionMap dict.Dict // key所对应的版本（uint32）
	// dict.Dict将确保其方法的并发安全
	// 仅对复杂的命令使用该互斥锁，如，rpush，incr,msetnx...
	locker *lock.Locks
	AddAof func(Cmdline)
}

// slava命令的执行函数
// args 中并不包括cmd命令行

type ExecFunc func(db *DB, args [][]byte) slava.Reply

// CmdLine表示命令行
// TODO cmd统一
type CmdLine = [][]byte
type Cmdline = [][]byte

// PreFun 将在”multi“命令中使用，返回相关的写键和读键
// 该函数在ExecFunc前执行，负责分析命令行读写了哪些key便于进行加锁

type PreFunc func(args [][]byte) ([]string, []string)

//UndoFunc返回给定命令行的撤消日志，仅在事务中使用，负责undo logs以备事务执行过程中遇到错误需要回滚
//撤消时从头到尾执行

type UndoFunc func(db *DB, args [][]byte) []Cmdline

// 初始化DB
func MakeDB() *DB {
	return makeDB()
}

func makeDB() *DB {
	db := &DB{
		data:       dict.MakeConcurrent(DataDictSize),
		ttlMap:     dict.MakeConcurrent(TtlDictSize),
		versionMap: dict.MakeConcurrent(DataDictSize),
		locker:     lock.Make(LockerSize),
		AddAof:     func(line Cmdline) {},
	}
	return db
}

// 一个分数据库中执行命令
// transaction包下有控制事务命令和其他不能在事务中执行的命令

func (db *DB) Exec(c slava.Connection, cmdLine Cmdline) slava.Reply {
	cmdName := strings.ToLower(string(cmdLine[0]))
	if cmdName == "multi" { // 命令为开始事务的命令
		if len(cmdLine) != 1 {
			return protocol.MakeArgNumErrReply(cmdName)
		}
		return StartMulti(c)
	} else if cmdName == "discard" { // 撤销事务
		if len(cmdLine) != 1 {
			return protocol.MakeArgNumErrReply(cmdName)
		}
		return DiscardMulti(c)
	} else if cmdName == "exec" { // 提交事务
		if len(cmdLine) != 1 {
			return protocol.MakeErrReply(cmdName)
		}
		return execMulti(db, c)
	} else if cmdName == "watch" {
		if !validateArity(-2, cmdLine) {
			return protocol.MakeArgNumErrReply(cmdName)
		}
		return Watch(db, c, cmdLine[1:])
	}
	// 如果现在是开启事务的状态，则将该命令入队
	if c != nil && c.InMultiState() {
		return EnqueueCmd(c, cmdLine)
	}
	// 如果不是事务的状态，则是普通命令
	return db.execNormalCommand(cmdLine)
}

// 执行普通的命令
func (db *DB) execNormalCommand(cmdLine [][]byte) slava.Reply {
	cmdName := strings.ToLower(string(cmdLine[0]))
	cmd, ok := cmdTable[cmdName]
	if !ok {
		return protocol.MakeErrReply("ERR unknown command '" + cmdName + "'")
	}
	if !validateArity(cmd.arity, cmdLine) {
		return protocol.MakeArgNumErrReply(cmdName)
	}
	prepare := cmd.prepare
	write, read := prepare(cmdLine[1:])
	db.addVersion(write...)
	db.locker.RWLocks(write, read)
	defer db.locker.RWUnLocks(write, read)
	fun := cmd.executor
	return fun(db, cmdLine[1:])
}

func validateArity(arity int, cmdArgs [][]byte) bool {
	argNum := len(cmdArgs)
	if arity >= 0 {
		return arity == argNum
	}
	return argNum >= -arity
}

// GetVersion返回某个给定key的版本码，用在watch中，在执行exec操作的时候用来对比监控的key值是否发生改变

func (db *DB) GetVersion(key string) uint32 {
	entity, ok := db.versionMap.Get(key)
	if !ok { // 没取到值
		return 0
	}
	return entity.(uint32)
}

// execWithLock executes normal commands, invoker should provide locks
func (db *DB) execWithLock(cmdLine [][]byte) slava.Reply {
	cmdName := strings.ToLower(string(cmdLine[0]))
	cmd, ok := cmdTable[cmdName]
	if !ok {
		return protocol.MakeErrReply("ERR unknown command '" + cmdName + "'")
	}
	if !validateArity(cmd.arity, cmdLine) {
		return protocol.MakeArgNumErrReply(cmdName)
	}
	fun := cmd.executor
	return fun(db, cmdLine[1:])
}

// 对于写入的操作，则要更改相应键值的version信息
func (db *DB) addVersion(keys ...string) {
	for _, key := range keys {
		versionCode := db.GetVersion(key)
		db.versionMap.Put(key, versionCode+1)
	}
}

// TTL的函数
func genExpireTask(key string) string {
	return "expire:" + key
}

// Expire 设置key的过期时间
func (db *DB) Expire(key string, expireTime time.Time) {
	db.ttlMap.Put(key, expireTime)
	taskKey := genExpireTask(key)
	timewheel.At(expireTime, taskKey, func() {
		keys := []string{key}
		db.locker.RWLocks(keys, nil)
		defer db.locker.RWUnLocks(keys, nil)
		// check-lock-check, ttl may be updated during waiting lock
		logger.Info("expire" + key)
		rawExpireTime, ok := db.ttlMap.Get(key)
		if !ok {
			return
		}
		expireTime, _ := rawExpireTime.(time.Time)
		expired := time.Now().After(expireTime)
		if expired {
			db.Remove(key)
		}
	})
}

// 取消一个key的ttlCmd

func (db *DB) Persist(key string) {
	db.ttlMap.Remove(key)
	taskKey := genExpireTask(key)
	timewheel.Cancel(taskKey)
}

// 判断一个键值是否过期
func (db *DB) IsExpired(key string) bool {
	//db.locker.Lock(key)
	//defer db.locker.UnLock(key)
	rawExpireTime, exists := db.ttlMap.Get(key)
	if !exists {
		return false
	}
	expireTime, _ := rawExpireTime.(time.Time)
	expired := time.Now().After(expireTime)
	if expired {
		db.Remove(key)
	}
	return expired
}

// 实现分数据库对数据的操作
func (db *DB) GetEntity(key string) (*database.DataEntity, bool) {
	raw, exists := db.data.Get(key)
	if !exists {
		return nil, false
	}
	if db.IsExpired(key) {
		return nil, false
	}
	entity, _ := raw.(*database.DataEntity)
	atomic.StoreInt32(&entity.Lru, int32(time.Now().Unix()))
	atomic.AddUint32(&entity.Lfu, 1)
	return entity, true
}

// 在分数据库中加入一个key value
func (db *DB) PutEntity(key string, entity *database.DataEntity) int {
	atomic.StoreInt32(&entity.Lru, int32(time.Now().Unix()))
	atomic.AddUint32(&entity.Lfu, 1)
	return db.data.Put(key, entity)
}

// 如果存在则修改，返回1，如果不存在返回0
func (db *DB) PutIfAbsent(key string, entity *database.DataEntity) int {
	atomic.StoreInt32(&entity.Lru, int32(time.Now().Unix()))
	atomic.AddUint32(&entity.Lfu, 1)
	return db.data.PutIfAbsent(key, entity)
}

// 如果不存在的时候加入，返回1，否则返回0
func (db *DB) PutIfExists(key string, entity *database.DataEntity) int {
	atomic.StoreInt32(&entity.Lru, int32(time.Now().Unix()))
	atomic.AddUint32(&entity.Lfu, 1)
	return db.data.PutIfExists(key, entity)
}

// 从数据库中移除给定的key
func (db *DB) Remove(key string) {
	db.data.Remove(key)
	db.ttlMap.Remove(key)
	taskKey := genExpireTask(key)
	timewheel.Cancel(taskKey)
}

// 从数据库中移除多个键值

func (db *DB) Removes(keys ...string) int {
	deleted := 0
	for _, key := range keys {
		_, ok := db.GetEntity(key)
		if ok {
			db.Remove(key)
			deleted++
		}
	}
	return deleted
}

// Flush清空数据库
func (db *DB) Flush() {
	*db = *makeDB()
}

// 遍历所有的key

func (db *DB) ForEach(cb func(key string, data *database.DataEntity, expiration *time.Time) bool) {
	db.data.ForEach(func(key string, val interface{}) bool {
		entity, _ := val.(*database.DataEntity)
		rawExpireTime, ok := db.ttlMap.Get(key)
		var expiration *time.Time
		if ok {
			expireTime := rawExpireTime.(time.Time)
			expiration = &expireTime
		}
		return cb(key, entity, expiration)
	})
}

func (db *DB) GetAsString(key string) ([]byte, protocol.ErrorReply) {
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

/* ---- Lock Function ----- */

// RWLocks lock keys for writing and reading
func (db *DB) RWLocks(writeKeys []string, readKeys []string) {
	db.locker.RWLocks(writeKeys, readKeys)
}

// RWUnLocks unlock keys for writing and reading
func (db *DB) RWUnLocks(writeKeys []string, readKeys []string) {
	db.locker.RWUnLocks(writeKeys, readKeys)
}
