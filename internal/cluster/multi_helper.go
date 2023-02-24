package cluster

import (
	"bytes"
	"encoding/base64"
	"slava/internal/aof"

	"slava/internal/protocol"
	"slava/internal/slava/parser"
)

func encodeCmdLine(cmdLines []aof.CmdLine) [][]byte {
	var result [][]byte
	for _, line := range cmdLines {
		raw := protocol.MakeMultiBulkReply(line).ToBytes()
		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(raw)))
		base64.StdEncoding.Encode(encoded, raw)
		result = append(result, encoded)
	}
	return result
}

func parseEncodedMultiRawReply(args [][]byte) (*protocol.MultiRawReply, error) {
	cmdBuf := new(bytes.Buffer)
	for _, arg := range args {
		dbuf := make([]byte, base64.StdEncoding.DecodedLen(len(arg)))
		n, err := base64.StdEncoding.Decode(dbuf, arg)
		if err != nil {
			continue
		}
		cmdBuf.Write(dbuf[:n])
	}
	cmds, err := parser.ParseBytes(cmdBuf.Bytes())
	if err != nil {
		return nil, protocol.MakeErrReply(err.Error())
	}
	return protocol.MakeMultiRawReply(cmds), nil
}

func encodeMultiRawReply(src *protocol.MultiRawReply) *protocol.MultiBulkReply {
	args := make([][]byte, 0, len(src.Replies))
	for _, rep := range src.Replies {
		raw := rep.ToBytes()
		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(raw)))
		base64.StdEncoding.Encode(encoded, raw)
		args = append(args, encoded)
	}
	return protocol.MakeMultiBulkReply(args)
}
