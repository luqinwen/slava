package database

import "slava/internal/utils"

func readFirstKey(args [][]byte) ([]string, []string) {
	key := string(args[0])
	return nil, []string{key}
}

func writeFirstKey(args [][]byte) ([]string, []string) {
	key := string(args[0])
	return []string{key}, nil
}

func readAllKeys(args [][]byte) ([]string, []string) {
	keys := make([]string, len(args))
	for i := 0; i < len(args); i++ {
		keys[i] = string(args[i])
	}
	return nil, keys
}

func noPrepare(args [][]byte) ([]string, []string) {
	return nil, nil
}

func rollbackFirstKey(db *DB, args [][]byte) []Cmdline {
	key := string(args[0])
	return rollbackGivenKeys(db, key)
}

func rollbackGivenKeys(db *DB, keys ...string) []Cmdline {
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
