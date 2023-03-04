package database

import (
	"slava/internal/interface/slava"
	"time"
)

// CmdLine is alias for [][]byte, represents a command line
type CmdLine = [][]byte

// DB is the interface for slava style storage engine
type DB interface {
	Exec(client slava.Connection, cmdLine [][]byte) slava.Reply
	AfterClientClose(c slava.Connection)
	Close()
}

// DBEngine is the embedding storage engine exposing more methods for complex application
type DBEngine interface {
	DB
	ExecWithLock(conn slava.Connection, cmdLine [][]byte) slava.Reply
	ExecMulti(conn slava.Connection, watching map[string]uint32, cmdLines []CmdLine) slava.Reply
	GetUndoLogs(dbIndex int, cmdLine [][]byte) []CmdLine
	ForEach(dbIndex int, cb func(key string, data *DataEntity, expiration *time.Time) bool)
	RWLocks(dbIndex int, writeKeys []string, readKeys []string)
	RWUnLocks(dbIndex int, writeKeys []string, readKeys []string)
	GetDBSize(dbIndex int) (int, int)
}

// DataEntity stores data bound to a key, including a string, list, hash, set and so on
type DataEntity struct {
	Data interface{}
	Lru  int32  // time unix
	Lfu  uint32 // count
}
