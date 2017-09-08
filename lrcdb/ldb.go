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

package lrcdb

import (
	"github.com/syndtr/goleveldb/leveldb"
	"sync"
	"github.com/Loopring/ringminer/log"
	"github.com/syndtr/goleveldb/leveldb/opt"
	"github.com/syndtr/goleveldb/leveldb/iterator"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type LDB struct {
	db *leveldb.DB
	batch *leveldb.Batch
	lock sync.RWMutex
}

var OpenFileLimit = 64

type LDBDatabase struct {
	fn string            // filename for reporting
	db *leveldb.DB       // LevelDB instance
	lock sync.Mutex      // Mutex protecting the quit channel access
}

// TODO(fk): use config and log
func NewDB(file string, cache, handles int) *LDBDatabase {
	l := &LDBDatabase{}

	// Ensure we have some minimal caching and file guarantees
	if cache < 8 {
		cache = 8
	}
	if handles < 4 {
		handles = 4
	}

	// log.Info("Allocated cache and file handles", cache)

	// Open the db and recover any potential corruptions
	db, err := leveldb.OpenFile(file, &opt.Options{
		OpenFilesCacheCapacity: handles,
		BlockCacheCapacity:     cache * opt.MiB,
		WriteBuffer:            cache * opt.MiB, // Two of these are used internally
	})
	if err != nil {
		log.Crit(log.ERROR_LDB_CREATE_FAILED, err.Error())
	}

	l.db = db
	l.fn = file

	// TODO(fk): implement recovery

	// TODO(fk): (Re)check for errors and abort if opening of the db failed

	return l
}

func (db *LDBDatabase) Path() string {
	return db.fn
}

func (db *LDBDatabase) Put(key []byte, value []byte) error {
	return db.db.Put(key, value, nil)
}

func (db *LDBDatabase) Get(key []byte) ([]byte, error) {
	dat, err := db.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	return dat, nil
}

func (db *LDBDatabase) Delete(key []byte) error {
	return db.db.Delete(key, nil)
}

func (db *LDBDatabase) NewIterator() iterator.Iterator {
	return db.db.NewIterator(nil, nil)
}

// TODO(fk): scan db with iterator means nothing
func (db *LDBDatabase) Scan() {
	iter := db.db.NewIterator(nil, nil)
	for iter.Next() {
		iter.Key()
		iter.Value()
	}
}

// Range create prefix
func Range() {
	prefix := []byte("")
	util.BytesPrefix(prefix)
}

func (db *LDBDatabase) Close() {
	db.lock.Lock()
	defer db.lock.Unlock()

	err := db.db.Close()
	if err == nil {
		log.Info("Database closed", err.Error())
	} else {
		log.Error("Failed to close database", err.Error())
	}
}

func (db *LDBDatabase) LDB() *leveldb.DB {
	return db.db
}

func (db *LDBDatabase) NewBatch() Batch {
	return &ldbBatch{db: db.db, b: new(leveldb.Batch)}
}

type ldbBatch struct {
	db *leveldb.DB
	b  *leveldb.Batch
}

func (b *ldbBatch) Put(key, value []byte) error {
	b.b.Put(key, value)
	return nil
}

func (b *ldbBatch) Write() error {
	return b.db.Write(b.b, nil)
}

type table struct {
	db     Database
	prefix string
}

// NewTable returns a Database object that prefixes all keys with a given
// string.
func NewTable(db Database, prefix string) Database {
	return &table{
		db:     db,
		prefix: prefix,
	}
}

func (dt *table) Put(key []byte, value []byte) error {
	return dt.db.Put(append([]byte(dt.prefix), key...), value)
}

func (dt *table) Get(key []byte) ([]byte, error) {
	return dt.db.Get(append([]byte(dt.prefix), key...))
}

func (dt *table) Delete(key []byte) error {
	return dt.db.Delete(append([]byte(dt.prefix), key...))
}

func (dt *table) Close() {
	// Do nothing; don't close the underlying DB.
}

type tableBatch struct {
	batch  Batch
	prefix string
}

// NewTableBatch returns a Batch object which prefixes all keys with a given string.
func NewTableBatch(db Database, prefix string) Batch {
	return &tableBatch{db.NewBatch(), prefix}
}

func (dt *table) NewBatch() Batch {
	return &tableBatch{dt.db.NewBatch(), dt.prefix}
}

func (tb *tableBatch) Put(key, value []byte) error {
	return tb.batch.Put(append([]byte(tb.prefix), key...), value)
}

func (tb *tableBatch) Write() error {
	return tb.batch.Write()
}
