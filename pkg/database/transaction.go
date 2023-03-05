package database

import (
	"slava/internal/interface/slava"
	"slava/internal/protocol"
	"strings"
)

// StartMulti 开启事务
func StartMulti(conn slava.Connection) slava.Reply {
	if conn.InMultiState() {
		return protocol.MakeErrReply("ERR MULTI calls can not be nested") // multi不能嵌套
	}
	conn.SetMultiState(true)
	return protocol.MakeOkReply()
}

// DiscardMulti 撤销事务，删除事务中的命令
func DiscardMulti(conn slava.Connection) slava.Reply {
	if !conn.InMultiState() {
		return protocol.MakeErrReply("ERR DISCARD without MULTI")
	}
	// 清除multi中的命令
	conn.ClearQueuedCmds()
	// 表示退出事务
	conn.SetMultiState(false)
	return protocol.MakeOkReply()
}

// exec提交事务
func execMulti(db *DB, conn slava.Connection) slava.Reply {
	if !conn.InMultiState() {
		return protocol.MakeErrReply("ERR EXEC without Multi")
	}
	defer conn.SetMultiState(false)
	if len(conn.GetTxErrors()) > 0 {
		return protocol.MakeErrReply("EXECABORT Transaction discarded because of previous errors.")
	}
	cmdLines := conn.GetQueuedCmdLine()
	return db.ExecMulti(conn, conn.GetWatching(), cmdLines)
}

// ExecMulti 以原子性和隔离性方式执行多命令事务
func (db *DB) ExecMulti(conn slava.Connection, watching map[string]uint32, cmdLines []CmdLine) slava.Reply {
	// prepare
	writeKeys := make([]string, 0, len(cmdLines)) // 提升性能，预分配内存
	readKeys := make([]string, 0, len(cmdLines))
	for _, cmdLine := range cmdLines {
		cmdName := strings.ToLower(string(cmdLine[0]))
		cmd := cmdTable[cmdName]
		prepare := cmd.prepare
		write, read := prepare(cmdLine[1:])
		writeKeys = append(writeKeys, write...)
		readKeys = append(readKeys, read...)
	}
	// set Watch
	watchingKeys := make([]string, 0, len(watching))
	for key := range watching {
		watchingKeys = append(watchingKeys, key)
	}
	// 将监控的key加入到readKeys的集合中，对writekeys和readkeys进行加锁，他们可能会存在相同的key
	readKeys = append(readKeys, watchingKeys...)
	db.locker.RWLocks(writeKeys, readKeys)
	defer db.locker.RWUnLocks(writeKeys, readKeys)
	// 如果watch keys 有发生改变，则回滚事务，并退出
	if isWatchingChanged(db, watching) { // 说明监控的key有别的线程修改过，则回滚
		return protocol.MakeEmptyMultiBulkReply()
	}
	// 如果监控的key没有被别的线程修改则执行其中的事务
	results := make([]slava.Reply, 0, len(cmdLines)) // 记录所有事务所有指令的返回结果
	aborted := false                                 // 取消事务，开始先默认false
	undoCmdLines := make([][]CmdLine, 0, len(cmdLines))
	for _, cmdLine := range cmdLines {
		undoCmdLines = append(undoCmdLines, db.GetUndoLogs(cmdLine))
		result := db.execWithLock(cmdLine)
		if protocol.IsErrorReply(result) {
			aborted = true // 如果事务执行出现错误，则aborted变成true
			// 执行出错的命令，不回滚
			undoCmdLines = undoCmdLines[:len(undoCmdLines)-1]
			break // 只要在事务中有一个命令行出错则停止循环
		}
		// 没有出错则将result加入到结果的数组中
		results = append(results, result)
	}
	// 如果中止标志部位true则返回结果，如果中止标志为true，则要进行回滚
	if !aborted {
		db.addVersion(writeKeys...)
		return protocol.MakeMultiRawReply(results)
	}
	// 如果种植条件aborted为true则要进行回滚
	size := len(undoCmdLines)
	for i := size - 1; i >= 0; i-- {
		curCmdLines := undoCmdLines[i]
		if len(curCmdLines) == 0 {
			continue
		}
		for _, cmdLine := range curCmdLines {
			db.execWithLock(cmdLine)
		}
	}
	return protocol.MakeErrReply("EXECABORT Transaction discarded because of previous errors.")
}

// Watch 监控key
func Watch(db *DB, conn slava.Connection, args [][]byte) slava.Reply {
	watching := conn.GetWatching()
	for _, bkey := range args {
		key := string(bkey)
		watching[key] = db.GetVersion(key)
	}
	return protocol.MakeOkReply()
}

func isWatchingChanged(db *DB, watching map[string]uint32) bool {
	for key, val := range watching {
		currentVersion := db.GetVersion(key)
		if currentVersion != val { // 说明修改过
			return true
		}
	}
	// 如果全部的key都没有修改过则返回false
	return false
}

func (db *DB) GetUndoLogs(cmdLine [][]byte) []CmdLine {
	cmdName := strings.ToLower(string(cmdLine[0]))
	cmd, ok := cmdTable[cmdName]
	if !ok {
		return nil
	}
	undo := cmd.undo
	if undo == nil {
		return nil
	}
	return undo(db, cmdLine[1:])
}

// EnqueueCmd 把属于事务的命令放入队列中
func EnqueueCmd(conn slava.Connection, cmdLine [][]byte) slava.Reply {
	cmdName := strings.ToLower(string(cmdLine[0]))
	cmd, ok := cmdTable[cmdName]
	if !ok {
		err := protocol.MakeErrReply("ERR unknown command '" + cmdName + "'")
		conn.AddTxError(err)
		return err
	}
	if cmd.prepare == nil {
		err := protocol.MakeErrReply("ERR command '" + cmdName + "' can not be used in MULTI")
		conn.AddTxError(err)
		return err
	}
	if !validateArity(cmd.arity, cmdLine) {
		err := protocol.MakeArgNumErrReply(cmdName)
		conn.AddTxError(err)
		return err
	}
	conn.EnqueueCmd(cmdLine)
	return protocol.MakeQueuedReply()
}

// GetRelatedKeys analysis related keys
func GetRelatedKeys(cmdLine [][]byte) ([]string, []string) {
	cmdName := strings.ToLower(string(cmdLine[0]))
	cmd, ok := cmdTable[cmdName]
	if !ok {
		return nil, nil
	}
	prepare := cmd.prepare
	if prepare == nil {
		return nil, nil
	}
	return prepare(cmdLine[1:])
}
