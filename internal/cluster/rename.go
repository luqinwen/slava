package cluster

import (
	"slava/internal/aof"
	"slava/internal/utils"
	"strconv"

	"slava/internal/interface/slava"
	"slava/internal/protocol"
)

// Rename renames a key, the origin and the destination must within the same node
func Rename(cluster *Cluster, c slava.Connection, args [][]byte) slava.Reply {
	if len(args) != 3 {
		return protocol.MakeErrReply("ERR wrong number of arguments for 'rename' command")
	}
	srcKey := string(args[1])
	destKey := string(args[2])
	srcNode := cluster.peerPicker.PickNode(srcKey)
	destNode := cluster.peerPicker.PickNode(destKey)
	if srcNode == destNode { // do fast
		return cluster.relay(srcNode, c, args)
	}
	groupMap := map[string][]string{
		srcNode:  {srcKey},
		destNode: {destKey},
	}
	txID := cluster.idGenerator.NextID()
	txIDStr := strconv.FormatInt(txID, 10)
	// prepare rename from
	srcPrepareResp := cluster.relayPrepare(srcNode, c, makeArgs("Prepare", txIDStr, "RenameFrom", srcKey))
	if protocol.IsErrorReply(srcPrepareResp) {
		// rollback src node
		requestRollback(cluster, c, txID, map[string][]string{srcNode: {srcKey}})
		return srcPrepareResp
	}
	srcPrepareMBR, ok := srcPrepareResp.(*protocol.MultiBulkReply)
	if !ok || len(srcPrepareMBR.Args) < 2 {
		requestRollback(cluster, c, txID, map[string][]string{srcNode: {srcKey}})
		return protocol.MakeErrReply("ERR invalid prepare response")
	}
	// prepare rename to
	destPrepareResp := cluster.relayPrepare(destNode, c, utils.ToCmdLine3("Prepare", []byte(txIDStr),
		[]byte("RenameTo"), []byte(destKey), srcPrepareMBR.Args[0], srcPrepareMBR.Args[1]))
	if protocol.IsErrorReply(destPrepareResp) {
		// rollback src node
		requestRollback(cluster, c, txID, groupMap)
		return destPrepareResp
	}
	if _, errReply := requestCommit(cluster, c, txID, groupMap); errReply != nil {
		requestRollback(cluster, c, txID, groupMap)
		return errReply
	}
	return protocol.MakeOkReply()
}

// prepareRenameFrom is prepare-function for RenameFrom, see prepareFuncMap
func prepareRenameFrom(cluster *Cluster, conn slava.Connection, cmdLine aof.CmdLine) slava.Reply {
	if len(cmdLine) != 2 {
		return protocol.MakeArgNumErrReply("RenameFrom")
	}
	key := string(cmdLine[1])
	existResp := cluster.db.ExecWithLock(conn, utils.ToCmdLine("Exists", key))
	if protocol.IsErrorReply(existResp) {
		return existResp
	}
	existIntResp := existResp.(*protocol.IntReply)
	if existIntResp.Code == 0 {
		return protocol.MakeErrReply("ERR no such key")
	}
	return cluster.db.ExecWithLock(conn, utils.ToCmdLine2("DumpKey", key))
}

func prepareRenameNxTo(cluster *Cluster, conn slava.Connection, cmdLine aof.CmdLine) slava.Reply {
	if len(cmdLine) != 4 {
		return protocol.MakeArgNumErrReply("RenameNxTo")
	}
	key := string(cmdLine[1])
	existResp := cluster.db.ExecWithLock(conn, utils.ToCmdLine("Exists", key))
	if protocol.IsErrorReply(existResp) {
		return existResp
	}
	existIntResp := existResp.(*protocol.IntReply)
	if existIntResp.Code == 1 {
		return protocol.MakeErrReply(keyExistsErr)
	}
	return protocol.MakeOkReply()
}

func init() {
	registerPrepareFunc("RenameFrom", prepareRenameFrom)
	registerPrepareFunc("RenameNxTo", prepareRenameNxTo)
}

// RenameNx renames a key, only if the new key does not exist.
// The origin and the destination must within the same node
func RenameNx(cluster *Cluster, c slava.Connection, args [][]byte) slava.Reply {
	if len(args) != 3 {
		return protocol.MakeErrReply("ERR wrong number of arguments for 'renamenx' command")
	}
	srcKey := string(args[1])
	destKey := string(args[2])
	srcNode := cluster.peerPicker.PickNode(srcKey)
	destNode := cluster.peerPicker.PickNode(destKey)
	if srcNode == destNode {
		return cluster.relay(srcNode, c, args)
	}
	groupMap := map[string][]string{
		srcNode:  {srcKey},
		destNode: {destKey},
	}
	txID := cluster.idGenerator.NextID()
	txIDStr := strconv.FormatInt(txID, 10)
	// prepare rename from
	srcPrepareResp := cluster.relayPrepare(srcNode, c, makeArgs("Prepare", txIDStr, "RenameFrom", srcKey))
	if protocol.IsErrorReply(srcPrepareResp) {
		// rollback src node
		requestRollback(cluster, c, txID, map[string][]string{srcNode: {srcKey}})
		return srcPrepareResp
	}
	srcPrepareMBR, ok := srcPrepareResp.(*protocol.MultiBulkReply)
	if !ok || len(srcPrepareMBR.Args) < 2 {
		requestRollback(cluster, c, txID, map[string][]string{srcNode: {srcKey}})
		return protocol.MakeErrReply("ERR invalid prepare response")
	}
	// prepare rename to
	destPrepareResp := cluster.relayPrepare(destNode, c, utils.ToCmdLine3("Prepare", []byte(txIDStr),
		[]byte("RenameNxTo"), []byte(destKey), srcPrepareMBR.Args[0], srcPrepareMBR.Args[1]))
	if protocol.IsErrorReply(destPrepareResp) {
		// rollback src node
		requestRollback(cluster, c, txID, groupMap)
		if re := destPrepareResp.(protocol.ErrorReply); re.Error() == keyExistsErr {
			return protocol.MakeIntReply(0)
		}
		return destPrepareResp
	}
	if _, errReply := requestCommit(cluster, c, txID, groupMap); errReply != nil {
		requestRollback(cluster, c, txID, groupMap)
		return errReply
	}
	return protocol.MakeIntReply(1)
}
