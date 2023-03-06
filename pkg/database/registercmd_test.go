package database

import (
	"strings"
	"testing"

	. "slava/internal/data"
)

var mockCmdTable = make(map[string]*command)
var mockDB = makeDB()

func RegisterCommandTest(name string, executor ExecFunc, prepare PreFunc, rollback UndoFunc, arity int, flags int) {
	name = strings.ToLower(name)
	mockCmdTable[name] = &command{
		executor: executor,
		prepare:  prepare,
		undo:     rollback,
		arity:    arity,
		flags:    flags,
	}
}

func TestRegisterCmd(t *testing.T) {
	RegisterCommandTest("Set", execSet, WriteFirstKey, RollbackFirstKey, ArityNegativeTree, FlagWrite)
	// setnx k v
	RegisterCommandTest("Get", execGet, ReadFirstKey, nil, ArityTwo, FlagReadOnly)

	if len(mockCmdTable) != 2 {
		t.Errorf("The length of mockCmdTable expected be 2, but %d got", len(mockCmdTable))
	}
}

func TestSet(t *testing.T) {
	RegisterCommandTest("Set", execSet, WriteFirstKey, RollbackFirstKey, ArityNegativeTree, FlagWrite)

	cmdSet := make([][]byte, 0)
	cmdStringSet := "set hello 111"
	for _, cmd := range strings.Split(cmdStringSet, " ") {
		cmdSet = append(cmdSet, []byte(cmd))
	}

	cmdGet := make([][]byte, 0)
	cmdStringGet := "get hello"
	for _, cmd := range strings.Split(cmdStringGet, " ") {
		cmdGet = append(cmdGet, []byte(cmd))
	}

	mockDB.execNormalCommand(cmdSet)
	res, _ := mockDB.getAsString("hello")

	if string(res) != "111" {
		t.Errorf("The value corresponding to the key Hello should be 111, but %s got", res)
	}

	if string(mockDB.execNormalCommand(cmdGet).ToBytes()[4:7]) != "111" {
		t.Errorf("The value corresponding to the key Hello should be 111")
	}

}
