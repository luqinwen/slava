package cluster

import (
	"slava/internal/aof"
	"strconv"

	"slava/config"
	"slava/internal/interface/slava"
	"slava/internal/protocol"
)

func ping(cluster *Cluster, c slava.Connection, cmdLine aof.CmdLine) slava.Reply {
	return cluster.db.Exec(c, cmdLine)
}

/*----- utils -------*/

func makeArgs(cmd string, args ...string) [][]byte {
	result := make([][]byte, len(args)+1)
	result[0] = []byte(cmd)
	for i, arg := range args {
		result[i+1] = []byte(arg)
	}
	return result
}

// return node -> writeKeys
func (cluster *Cluster) groupBy(keys []string) map[string][]string {
	result := make(map[string][]string)
	for _, key := range keys {
		peer := cluster.peerPicker.PickNode(key)
		group, ok := result[peer]
		if !ok {
			group = make([]string, 0)
		}
		group = append(group, key)
		result[peer] = group
	}
	return result
}

func execSelect(c slava.Connection, args [][]byte) slava.Reply {
	dbIndex, err := strconv.Atoi(string(args[1]))
	if err != nil {
		return protocol.MakeErrReply("ERR invalid DB index")
	}
	if dbIndex >= config.Properties.Databases || dbIndex < 0 {
		return protocol.MakeErrReply("ERR DB index is out of range")
	}
	c.SelectDB(dbIndex)
	return protocol.MakeOkReply()
}
