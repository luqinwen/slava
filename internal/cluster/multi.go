package cluster

import (
	"slava/internal/aof"
	"strconv"

	"slava/internal/interface/slava"
	"slava/internal/protocol"
	"slava/internal/utils"
	"slava/pkg/database"
)

const relayMulti = "_multi"
const innerWatch = "_watch"

var relayMultiBytes = []byte(relayMulti)

// cmdLine == []string{"exec"}
func execMulti(cluster *Cluster, conn slava.Connection, cmdLine aof.CmdLine) slava.Reply {
	if !conn.InMultiState() {
		return protocol.MakeErrReply("ERR EXEC without MULTI")
	}
	defer conn.SetMultiState(false)
	cmdLines := conn.GetQueuedCmdLine()

	// analysis related keys
	keys := make([]string, 0) // may contains duplicate
	for _, cl := range cmdLines {
		wKeys, rKeys := database.GetRelatedKeys(cl)
		keys = append(keys, wKeys...)
		keys = append(keys, rKeys...)
	}
	watching := conn.GetWatching()
	watchingKeys := make([]string, 0, len(watching))
	for key := range watching {
		watchingKeys = append(watchingKeys, key)
	}
	keys = append(keys, watchingKeys...)
	if len(keys) == 0 {
		// empty transaction or only `PING`s
		return cluster.db.ExecMulti(conn, watching, cmdLines)
	}
	groupMap := cluster.groupBy(keys)
	if len(groupMap) > 1 {
		return protocol.MakeErrReply("ERR MULTI commands transaction must within one slot in cluster mode")
	}
	var peer string
	// assert len(groupMap) == 1
	for p := range groupMap {
		peer = p
	}

	// out parser not support protocol.MultiRawReply, so we have to encode it
	if peer == cluster.self {
		return cluster.db.ExecMulti(conn, watching, cmdLines)
	}
	return execMultiOnOtherNode(cluster, conn, peer, watching, cmdLines)
}

func execMultiOnOtherNode(cluster *Cluster, conn slava.Connection, peer string, watching map[string]uint32, cmdLines []aof.CmdLine) slava.Reply {
	defer func() {
		conn.ClearQueuedCmds()
		conn.SetMultiState(false)
	}()
	relayCmdLine := [][]byte{ // relay it to executing node
		relayMultiBytes,
	}
	// watching commands
	var watchingCmdLine = utils.ToCmdLine(innerWatch)
	for key, ver := range watching {
		verStr := strconv.FormatUint(uint64(ver), 10)
		watchingCmdLine = append(watchingCmdLine, []byte(key), []byte(verStr))
	}
	relayCmdLine = append(relayCmdLine, encodeCmdLine([]aof.CmdLine{watchingCmdLine})...)
	relayCmdLine = append(relayCmdLine, encodeCmdLine(cmdLines)...)
	var rawRelayResult slava.Reply
	if peer == cluster.self {
		// this branch just for testing
		rawRelayResult = execRelayedMulti(cluster, conn, relayCmdLine)
	} else {
		rawRelayResult = cluster.relay(peer, conn, relayCmdLine)
	}
	if protocol.IsErrorReply(rawRelayResult) {
		return rawRelayResult
	}
	_, ok := rawRelayResult.(*protocol.EmptyMultiBulkReply)
	if ok {
		return rawRelayResult
	}
	relayResult, ok := rawRelayResult.(*protocol.MultiBulkReply)
	if !ok {
		return protocol.MakeErrReply("execute failed")
	}
	rep, err := parseEncodedMultiRawReply(relayResult.Args)
	if err != nil {
		return protocol.MakeErrReply(err.Error())
	}
	return rep
}

// execRelayedMulti execute relayed multi commands transaction
// cmdLine format: _multi watch-cmdLine base64ed-cmdLine
// result format: base64ed-protocol list
func execRelayedMulti(cluster *Cluster, conn slava.Connection, cmdLine aof.CmdLine) slava.Reply {
	if len(cmdLine) < 2 {
		return protocol.MakeArgNumErrReply("_exec")
	}
	decoded, err := parseEncodedMultiRawReply(cmdLine[1:])
	if err != nil {
		return protocol.MakeErrReply(err.Error())
	}
	var txCmdLines []aof.CmdLine
	for _, rep := range decoded.Replies {
		mbr, ok := rep.(*protocol.MultiBulkReply)
		if !ok {
			return protocol.MakeErrReply("exec failed")
		}
		txCmdLines = append(txCmdLines, mbr.Args)
	}
	watching := make(map[string]uint32)
	watchCmdLine := txCmdLines[0] // format: _watch key1 ver1 key2 ver2...
	for i := 2; i < len(watchCmdLine); i += 2 {
		key := string(watchCmdLine[i-1])
		verStr := string(watchCmdLine[i])
		ver, err := strconv.ParseUint(verStr, 10, 64)
		if err != nil {
			return protocol.MakeErrReply("watching command line failed")
		}
		watching[key] = uint32(ver)
	}
	rawResult := cluster.db.ExecMulti(conn, watching, txCmdLines[1:])
	_, ok := rawResult.(*protocol.EmptyMultiBulkReply)
	if ok {
		return rawResult
	}
	resultMBR, ok := rawResult.(*protocol.MultiRawReply)
	if !ok {
		return protocol.MakeErrReply("exec failed")
	}
	return encodeMultiRawReply(resultMBR)
}

func execWatch(cluster *Cluster, conn slava.Connection, args [][]byte) slava.Reply {
	if len(args) < 2 {
		return protocol.MakeArgNumErrReply("watch")
	}
	args = args[1:]
	watching := conn.GetWatching()
	for _, bkey := range args {
		key := string(bkey)
		peer := cluster.peerPicker.PickNode(key)
		result := cluster.relay(peer, conn, utils.ToCmdLine("GetVer", key))
		if protocol.IsErrorReply(result) {
			return result
		}
		intResult, ok := result.(*protocol.IntReply)
		if !ok {
			return protocol.MakeErrReply("get version failed")
		}
		watching[key] = uint32(intResult.Code)
	}
	return protocol.MakeOkReply()
}
