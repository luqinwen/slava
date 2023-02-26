package database

import (
	"fmt"
	"os"
	"slava/pkg/datastruct/quicklist"

	"github.com/hdt3213/rdb/core"
	rdb "github.com/hdt3213/rdb/parser"
	"slava/config"
	"slava/internal/aof"
	"slava/internal/interface/database"
	"slava/pkg/datastruct/dict"
	SortedSet "slava/pkg/datastruct/sortedset"
	"sync/atomic"
)

func (server *Server) loadRdbFile() error {
	rdbFile, err := os.Open(config.Properties.RDBFilename)
	if err != nil {
		return fmt.Errorf("open rdb file failed " + err.Error())
	}
	defer func() {
		_ = rdbFile.Close()
	}()
	decoder := rdb.NewDecoder(rdbFile)
	err = server.loadRDB(decoder)
	if err != nil {
		return fmt.Errorf("dump rdb file failed " + err.Error())
	}
	return nil
}

func (server *Server) loadRDB(dec *core.Decoder) error {
	return dec.Parse(func(o rdb.RedisObject) bool {
		db := server.mustSelectDB(o.GetDBIndex())
		var entity *database.DataEntity
		switch o.GetType() {
		case rdb.StringType:
			str := o.(*rdb.StringObject)
			entity = &database.DataEntity{
				Data: str.Value,
			}
		case rdb.ListType:
			listObj := o.(*rdb.ListObject)
			list := quicklist.NewQuickList()
			for _, v := range listObj.Values {
				list.Add(v)
			}
			entity = &database.DataEntity{
				Data: list,
			}
		case rdb.HashType:
			hashObj := o.(*rdb.HashObject)
			// Todo make concurrent的默认值
			hash := dict.MakeConcurrent(4)
			for k, v := range hashObj.Hash {
				hash.Put(k, v)
			}
			entity = &database.DataEntity{
				Data: hash,
			}
		case rdb.ZSetType:
			zsetObj := o.(*rdb.ZSetObject)
			zSet := SortedSet.Make()
			for _, e := range zsetObj.Entries {
				zSet.Add(e.Member, e.Score)
			}
			entity = &database.DataEntity{
				Data: zSet,
			}
		}
		if entity != nil {
			db.PutEntity(o.GetKey(), entity)
			if o.GetExpiration() != nil {
				db.Expire(o.GetKey(), *o.GetExpiration())
			}
			db.AddAof(aof.EntityToCmd(o.GetKey(), entity).Args)
		}
		return true
	})
}

func NewPersister(db database.DBEngine, filename string, load bool, fsync string) (*aof.Persister, error) {
	return aof.NewPersister(db, filename, load, fsync, func() database.DBEngine {
		return MakeAuxiliaryServer()
	})
}

func (server *Server) AddAof(dbIndex int, cmdLine aof.CmdLine) {
	if server.persister != nil {
		server.persister.SaveCmdLine(dbIndex, cmdLine)
	}
}

func (server *Server) bindPersister(aofHandler *aof.Persister) {
	server.persister = aofHandler
	// bind SaveCmdLine
	for _, db := range server.dbSet {
		singleDB := db.Load().(*DB)
		singleDB.AddAof = func(line aof.CmdLine) {
			if config.Properties.AppendOnly { // config may be changed during runtime
				server.persister.SaveCmdLine(singleDB.index, line)
			}
		}
	}
}

// MakeAuxiliaryServer create a Server only with basic capabilities for aof rewrite and other usages
func MakeAuxiliaryServer() *Server {
	mdb := &Server{}
	mdb.dbSet = make([]*atomic.Value, config.Properties.Databases)
	for i := range mdb.dbSet {
		holder := &atomic.Value{}
		//holder.Store(makeBasicDB())
		holder.Store(makeDB())
		mdb.dbSet[i] = holder
	}
	return mdb
}
