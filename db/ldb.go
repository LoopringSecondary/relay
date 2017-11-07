/*

  Copyright 2017 Loopring Project Ltd (Loopring Foundation).

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

  http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.

*/

package db

import (
	"github.com/Loopring/ringminer/config"
	"github.com/Loopring/ringminer/log"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/util"
	"math/big"
	"sync"
)

var OpenFileLimit = 64

type LDBDatabase struct {
	*leveldb.DB            // LevelDB instance
	fn          string     // filename for reporting
	lock        sync.Mutex // Mutex protecting the quit channel access
}

// TODO(fk): use config and log
func NewDB(dbOpts config.DbOptions) *LDBDatabase {
	l := &LDBDatabase{}

	cache := dbOpts.CacheCapacity
	handles := dbOpts.BufferCapacity
	// Ensure we have some minimal caching and file guarantees
	if cache < 8 {
		cache = 8
	}
	if handles < 4 {
		handles = 4
	}

	// log.Info("Allocated cache and file handles", cache)

	// Open the db and recover any potential corruptions
	db, err := leveldb.OpenFile(dbOpts.DataDir, &opt.Options{
		OpenFilesCacheCapacity: cache,
		BlockCacheCapacity:     cache * opt.MiB,
		WriteBuffer:            cache * opt.MiB, // Two of these are used internally
	})
	if err != nil {
		log.Fatalf("leveldb create failed:%s", err.Error())
	}

	l.DB = db
	l.fn = dbOpts.DataDir

	// TODO(fk): implement recovery

	// TODO(fk): (Re)check for errors and abort if opening of the db failed

	return l
}

func (db *LDBDatabase) Path() string {
	return db.fn
}

func (db *LDBDatabase) Put(key []byte, value []byte) error {
	return db.DB.Put(key, value, nil)
}

func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
	dat, err := db.DB.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

func (db *LDBDatabase) Delete(key []byte) error {
	return db.DB.Delete(key, nil)
}

func (db *LDBDatabase) NewIterator(start []byte, limit []byte) Iterator {
	return db.DB.NewIterator(&util.Range{Start: start, Limit: limit}, nil)
}

// TODO(fk): scan db with iterator means nothing
//func (db *LDBDatabase) Scan() (map[string]string, error) {
//	var data map[string]string
//	iter := db.DB.NewIterator(nil, nil)
//	for iter.Next() {
//		data[string(iter.Key())] = string(iter.Value())
//	}
//	return data, nil
//}

func (db *LDBDatabase) Close() {
	db.lock.Lock()
	defer db.lock.Unlock()

	err := db.DB.Close()
	if err == nil {
		log.Info("Database closed", log.NewField("content", err.Error()))
	} else {
		log.Error("Failed to close database", log.NewField("content", err.Error()))
	}
}

func (db *LDBDatabase) LDB() *leveldb.DB {
	return db.DB
}

func (db *LDBDatabase) NewBatch() Batch {
	return &ldbBatch{db: db.DB, Batch: new(leveldb.Batch)}
}

type ldbBatch struct {
	*leveldb.Batch
	db *leveldb.DB
}

func (b *ldbBatch) Write() error {
	return b.db.Write(b.Batch, nil)
}

const seprator = "_"

type table struct {
	Database
	prefix        []byte
	prefixLargest []byte //todo: remove it ??
}

// NewTable returns a Database object that prefixes all keys with a given string.
func NewTable(db Database, prefix string) Database {
	pb := []byte(prefix + seprator)
	bi := big.NewInt(0)
	bi.SetBytes(pb)
	bi.Add(bi, big.NewInt(1))

	return &table{
		Database:      db,
		prefix:        pb,
		prefixLargest: bi.Bytes(),
	}
}

func (dt *table) Put(key []byte, value []byte) error {
	return dt.Database.Put(append(dt.prefix, key...), value)
}

func (dt *table) Get(key []byte) ([]byte, error) {
	return dt.Database.Get(append(dt.prefix, key...))
}

func (dt *table) Delete(key []byte) error {
	return dt.Database.Delete(append(dt.prefix, key...))
}

func (dt *table) NewIterator(start []byte, limit []byte) Iterator {
	tableStart := append(dt.prefix, start...)
	tableLimit := append(dt.prefixLargest, limit...)
	return dt.Database.NewIterator(tableStart, tableLimit)
}

func (dt *table) Close() {
	// Do nothing; don't close the underlying DB.
}

func (dt *table) NewBatch() Batch {
	return &tableBatch{dt.NewBatch(), dt.prefix}
}

type tableBatch struct {
	Batch
	prefix []byte
}

// NewTableBatch returns a Batch object which prefixes all keys with a given string.
func NewTableBatch(db Database, prefix string) Batch {
	return &tableBatch{
		Batch:  db.NewBatch(),
		prefix: []byte(prefix + seprator),
	}
}

func (tb *tableBatch) Put(key, value []byte) {
	tb.Batch.Put(append(tb.prefix, key...), value)
}

func (tb *tableBatch) Delete(key []byte) {
	tb.Batch.Delete(append(tb.prefix, key...))
}
