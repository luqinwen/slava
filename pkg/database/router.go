package database

import (
	"strings"

	. "slava/internal/constData"
)

var cmdTable = make(map[string]*command)

type command struct {
	executor ExecFunc // 执行的函数
	prepare  PreFunc
	undo     UndoFunc
	arity    int // 表示输入的参数个数
	flags    int // 指示该只读还是写函数
}

//const (
//	flagWrite = iota
//	flagReadOnly
//)

// 注册函数
// 将命令注册到对应的cmdTable中

func RegisterCommand(name string, executor ExecFunc, prepare PreFunc, rollback UndoFunc, arity int, flags int) {
	name = strings.ToLower(name)
	cmdTable[name] = &command{
		executor: executor,
		prepare:  prepare,
		undo:     rollback,
		arity:    arity,
		flags:    flags,
	}
}

func isReadOnlyCommand(name string) bool {
	name = strings.ToLower(name)
	cmd := cmdTable[name]
	if cmd == nil {
		return false
	}
	return cmd.flags == FlagReadOnly
}
