package database

import (
	"strconv"

	"slava/internal/interface/slava"
	"slava/internal/protocol"
	"slava/internal/utils"
)

// 获取链表长度
func execListLen(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])

	list, errReply := db.getAsList(key)
	if errReply != nil {
		return errReply
	}
	if list == nil {
		return protocol.MakeIntReply(0)
	}

	size := list.Len()
	return protocol.MakeIntReply(int64(size))
}

// 执行LPush，并进行aof操作
func execListRPush(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	values := args[1:]

	list, _, errReply := db.getOrInitList(key)
	if errReply != nil {
		return errReply
	}

	for _, value := range values {
		list.RPush(string(value))
	}

	db.AddAof(utils.ToCmdLine3("rpush", args...))
	return protocol.MakeIntReply(int64(list.Len()))
}

// 通过ListRpop操作进行撤销
func undoRPush(db *DB, args [][]byte) []CmdLine {
	// args[0] key
	// args[1:] 参数
	key := string(args[0])
	count := len(args) - 1
	cmdLines := make([]CmdLine, 0, count)
	for i := 0; i < count; i++ {
		cmdLines = append(cmdLines, utils.ToCmdLine("ListRpop", key))
	}
	return cmdLines
}

// 执行LPush，并进行aof操作
func execListLPush(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	values := args[1:]

	list, _, errReply := db.getOrInitList(key)
	if errReply != nil {
		return errReply
	}

	for _, value := range values {
		list.LPush(string(value))
	}

	db.AddAof(utils.ToCmdLine3("lpush", args...))
	return protocol.MakeIntReply(int64(list.Len()))
}

func undoLPush(db *DB, args [][]byte) []CmdLine {
	// args[0] key
	// args[1:] 参数
	key := string(args[0])
	count := len(args) - 1
	cmdLines := make([]CmdLine, 0, count)
	for i := 0; i < count; i++ {
		cmdLines = append(cmdLines, utils.ToCmdLine("ListLpop", key))
	}
	return cmdLines
}

func execListRpop(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])

	list, errReply := db.getAsList(key)
	if errReply != nil {
		return errReply
	}
	if list == nil {
		return &protocol.NullBulkReply{}
	}

	node := list.RPop()

	db.AddAof(utils.ToCmdLine3("rpop", args...))
	return protocol.MakeBulkReply([]byte(node.GetValue()))
}

// 执行Lpop，并进行aof操作
func execListLpop(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])

	list, errReply := db.getAsList(key)
	if errReply != nil {
		return errReply
	}
	if list == nil {
		return &protocol.NullBulkReply{}
	}

	node := list.LPop()

	db.AddAof(utils.ToCmdLine3("lpop", args...))
	return protocol.MakeBulkReply([]byte(node.GetValue()))
}

func execListGetByIndex(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	index64, err := strconv.ParseInt(string(args[1]), 10, 64)
	if err != nil {
		return protocol.MakeErrReply("index64 is wrong")
	}
	index := int(index64)

	list, errReply := db.getAsList(key)
	if errReply != nil {
		return errReply
	}
	if list == nil {
		return &protocol.NullBulkReply{}
	}

	size := list.Len()
	// 超出list范围
	if index < -1*size || index >= size {
		return &protocol.NullBulkReply{}
	}

	node := list.GetByIndex(index)
	return protocol.MakeBulkReply([]byte(node.GetValue()))
}

func execListRange(db *DB, args [][]byte) slava.Reply {
	key := string(args[0])
	start64, err := strconv.ParseInt(string(args[1]), 10, 64)
	if err != nil {
		return protocol.MakeErrReply("start64 is wrong")
	}
	stop64, err := strconv.ParseInt(string(args[2]), 10, 64)
	if err != nil {
		return protocol.MakeErrReply("stop64 is wrong")
	}
	start := int(start64)
	stop := int(stop64)

	list, errReply := db.getAsList(key)
	if errReply != nil {
		return errReply
	}
	if list == nil {
		return &protocol.NullBulkReply{}
	}

	size := list.Len()
	if start < 0 || stop >= size {
		return &protocol.NullBulkReply{}
	} else if stop < start {
		return protocol.MakeErrReply("list range: stop < start")
	}

	nodes := list.Range(start, stop)
	result := make([][]byte, len(nodes))
	for i, node := range nodes {
		byte := []byte(node.GetValue())
		result[i] = byte
	}
	return protocol.MakeMultiBulkReply(result)
}

// 注册函数
// func RegisterCommand(name string, executor ExecFunc, prepare PreFunc, rollback UndoFunc, arity int, flags int)

// 读操作：无undo函数
// 写操作：有undo函数
func init() {
	RegisterCommand("ListLen", execListLen, ReadFirstKey, nil, 1, flagReadOnly)
	RegisterCommand("ListRPush", execListRPush, WriteFirstKey, undoRPush, 1, flagWrite)
	RegisterCommand("ListLPush", execListLPush, WriteFirstKey, undoLPush, 1, flagWrite)
	RegisterCommand("ListRpop", execListRpop, ReadFirstKey, nil, 1, flagReadOnly)
	RegisterCommand("ListLpop", execListLpop, ReadFirstKey, nil, 1, flagReadOnly)
	RegisterCommand("ListGetByIndex", execListGetByIndex, ReadFirstKey, nil, 1, flagReadOnly)
	RegisterCommand("ListRange", execListRange, ReadFirstKey, nil, 1, flagReadOnly)
}
