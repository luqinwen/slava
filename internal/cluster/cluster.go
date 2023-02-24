package cluster

import (
	"fmt"
	"runtime/debug"
	"slava/internal/aof"
	"strings"

	"slava/config"
	"slava/internal/interface/slava"
	"slava/internal/protocol"
	"slava/internal/slava/client"
	"slava/internal/utils"
	"slava/pkg/consistenthash"
	"slava/pkg/database"
	"slava/pkg/datastruct/dict"
	"slava/pkg/idgenerator"
	"slava/pkg/logger"
	"slava/pkg/pool"

	db "slava/internal/interface/database"
)

const (
	replicas = 4
)

type PeerPicker interface {
	AddNode(keys ...string)
	PickNode(key string) string
}

// Cluster represents a node of godis cluster
// it holds part of data and coordinates other nodes to finish transactions
type Cluster struct {
	self string

	nodes           []string
	peerPicker      PeerPicker
	nodeConnections map[string]*pool.Pool

	db           db.DBEngine
	transactions *dict.ConcurrentDict // id -> Transaction

	idGenerator *idgenerator.IDGenerator
	// use a variable to allow injecting stub for testing
	relayImpl func(cluster *Cluster, node string, c slava.Connection, cmdLine aof.CmdLine) slava.Reply
}

// if only one node involved in a transaction, just execute the command don't apply tcc procedure
var allowFastTransaction = true

// MakeCluster creates and starts a node of cluster
func MakeCluster() *Cluster {
	cluster := &Cluster{
		self: config.Properties.Self,

		db: database.NewStandaloneServer(),
		// TODO MakeConcurrent的默认值应该在config里面规定
		transactions:    dict.MakeConcurrent(replicas),
		peerPicker:      consistenthash.New(replicas, nil),
		nodeConnections: make(map[string]*pool.Pool),

		idGenerator: idgenerator.MakeGenerator(config.Properties.Self),
		relayImpl:   defaultRelayImpl,
	}
	contains := make(map[string]struct{})
	nodes := make([]string, 0, len(config.Properties.Peers)+1)
	for _, peer := range config.Properties.Peers {
		if _, ok := contains[peer]; ok {
			continue
		}
		contains[peer] = struct{}{}
		nodes = append(nodes, peer)
	}
	nodes = append(nodes, config.Properties.Self)
	cluster.peerPicker.AddNode(nodes...)
	connectionPoolConfig := pool.Config{
		MaxIdle:   1,
		MaxActive: 16,
	}
	for _, p := range config.Properties.Peers {
		peer := p
		factory := func() (interface{}, error) {
			c, err := client.MakeClient(peer)
			if err != nil {
				return nil, err
			}
			c.Start()
			// all peers of cluster should use the same password
			if config.Properties.RequirePass != "" {
				c.Send(utils.ToCmdLine("AUTH", config.Properties.RequirePass))
			}
			return c, nil
		}
		finalizer := func(x interface{}) {
			cli, ok := x.(client.Client)
			if !ok {
				return
			}
			cli.Close()
		}
		cluster.nodeConnections[peer] = pool.New(factory, finalizer, connectionPoolConfig)
	}
	cluster.nodes = nodes
	return cluster
}

// CmdFunc represents the handler of a slava command
type CmdFunc func(cluster *Cluster, c slava.Connection, cmdLine aof.CmdLine) slava.Reply

// Close stops current node of cluster
func (cluster *Cluster) Close() {
	cluster.db.Close()
	for _, pool := range cluster.nodeConnections {
		pool.Close()
	}
}

var router = makeRouter()

func isAuthenticated(c slava.Connection) bool {
	if config.Properties.RequirePass == "" {
		return true
	}
	return c.GetPassword() == config.Properties.RequirePass
}

// Exec executes command on cluster
func (cluster *Cluster) Exec(c slava.Connection, cmdLine [][]byte) (result slava.Reply) {
	defer func() {
		if err := recover(); err != nil {
			logger.Warn(fmt.Sprintf("error occurs: %v\n%s", err, string(debug.Stack())))
			result = &protocol.UnknownErrReply{}
		}
	}()
	cmdName := strings.ToLower(string(cmdLine[0]))
	if cmdName == "auth" {
		return database.Auth(c, cmdLine[1:])
	}
	if !isAuthenticated(c) {
		return protocol.MakeErrReply("NOAUTH Authentication required")
	}

	if cmdName == "multi" {
		if len(cmdLine) != 1 {
			return protocol.MakeArgNumErrReply(cmdName)
		}
		return database.StartMulti(c)
	} else if cmdName == "discard" {
		if len(cmdLine) != 1 {
			return protocol.MakeArgNumErrReply(cmdName)
		}
		return database.DiscardMulti(c)
	} else if cmdName == "exec" {
		if len(cmdLine) != 1 {
			return protocol.MakeArgNumErrReply(cmdName)
		}
		return execMulti(cluster, c, nil)
	} else if cmdName == "select" {
		if len(cmdLine) != 2 {
			return protocol.MakeArgNumErrReply(cmdName)
		}
		return execSelect(c, cmdLine)
	}
	if c != nil && c.InMultiState() {
		return database.EnqueueCmd(c, cmdLine)
	}
	cmdFunc, ok := router[cmdName]
	if !ok {
		return protocol.MakeErrReply("ERR unknown command '" + cmdName + "', or not supported in cluster mode")
	}
	result = cmdFunc(cluster, c, cmdLine)
	return
}

// AfterClientClose does some clean after client close connection
func (cluster *Cluster) AfterClientClose(c slava.Connection) {
	cluster.db.AfterClientClose(c)
}
