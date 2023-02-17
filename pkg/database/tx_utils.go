package database

import (
	"slava/internal/utils"
	"strconv"
)

func ReadFirstKey(args [][]byte) ([]string, []string) {
	key := string(args[0])
	return nil, []string{key}
}

func WriteFirstKey(args [][]byte) ([]string, []string) {
	key := string(args[0])
	return []string{key}, nil
}

func ReadAllKeys(args [][]byte) ([]string, []string) {
	keys := make([]string, len(args))
	for i := 0; i < len(args); i++ {
		keys[i] = string(args[i])
	}
	return nil, keys
}

//func noPrepare(args [][]byte) ([]string, []string) {
//	return nil, nil
//}

func RollbackFirstKey(db *DB, args [][]byte) []Cmdline {
	key := string(args[0])
	return RollbackGivenKeys(db, key)
}

func RollbackGivenKeys(db *DB, keys ...string) []Cmdline {
	var undoCmdLine [][][]byte
	for _, key := range keys {
		_, ok := db.GetEntity(key)
		if !ok {
			undoCmdLine = append(undoCmdLine, utils.ToCmdLine("DEL", key))
		} else {
			// TODO
		}
	}
	return undoCmdLine
}

func RollbackZSetFields(db *DB, key string, fields ...string) []Cmdline {
	var undoCmdLines [][][]byte
	zset, errReply := db.getAsSortedSet(key)
	if errReply != nil {
		return nil
	}
	if zset == nil {
		undoCmdLines = append(undoCmdLines,
			utils.ToCmdLine("DEL", key),
		)
		return undoCmdLines
	}
	for _, field := range fields {
		elem, ok := zset.Get(field)
		if !ok {
			undoCmdLines = append(undoCmdLines,
				utils.ToCmdLine("ZREM", key, field),
			)
		} else {
			score := strconv.FormatFloat(elem.Score, 'f', -1, 64)
			undoCmdLines = append(undoCmdLines,
				utils.ToCmdLine("ZADD", key, score, field),
			)
		}
	}
	return undoCmdLines
}
