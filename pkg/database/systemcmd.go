package database

import (
	"slava/config"
	"slava/internal/interface/slava"
	"slava/internal/protocol"
)

// Ping the server
func Ping(c slava.Connection, args [][]byte) slava.Reply {
	if len(args) == 0 {
		return &protocol.PongReply{}
	} else if len(args) == 1 {
		return protocol.MakeStatusReply(string(args[0]))
	} else {
		return protocol.MakeErrReply("ERR wrong number of arguments for 'ping' command")
	}
}

// Auth validate client's password
func Auth(c slava.Connection, args [][]byte) slava.Reply {
	if len(args) != 1 {
		return protocol.MakeErrReply("ERR wrong number of arguments for 'auth' command")
	}
	if config.Properties.RequirePass == "" {
		return protocol.MakeErrReply("ERR Client sent AUTH, but no password is set")
	}
	passwd := string(args[0])
	c.SetPassword(passwd)
	if config.Properties.RequirePass != passwd {
		return protocol.MakeErrReply("ERR invalid password")
	}
	return &protocol.OkReply{}
}

func isAuthenticated(c slava.Connection) bool {
	if config.Properties.RequirePass == "" {
		return true
	}
	return c.GetPassword() == config.Properties.RequirePass
}
