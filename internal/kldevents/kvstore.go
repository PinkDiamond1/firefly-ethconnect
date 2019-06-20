// Copyright 2019 Kaleido

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package kldevents

import (
	"fmt"

	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

type kvIterator interface {
	Key() string
	Value() []byte
	Next() bool
	Release()
}

type kvStore interface {
	Put(key string, val []byte) error
	Get(key string) ([]byte, error)
	Delete(key string) error
	NewIterator() kvIterator
	Close()
}

type levelDBKeyValueStore struct {
	db *leveldb.DB
}

func (k *levelDBKeyValueStore) Put(key string, val []byte) error {
	return k.db.Put([]byte(key), val, nil)
}

func (k *levelDBKeyValueStore) Get(key string) ([]byte, error) {
	return k.db.Get([]byte(key), nil)
}

func (k *levelDBKeyValueStore) Delete(key string) error {
	return k.db.Delete([]byte(key), nil)
}

func (k *levelDBKeyValueStore) NewIterator() kvIterator {
	return &levelDBKeyIterator{
		i: k.db.NewIterator(nil, nil),
	}
}

type levelDBKeyIterator struct {
	i iterator.Iterator
}

func (k *levelDBKeyIterator) Key() string {
	return string(k.i.Key())
}

func (k *levelDBKeyIterator) Value() []byte {
	return k.i.Value()
}

func (k *levelDBKeyIterator) Next() bool {
	return k.i.Next()
}

func (k *levelDBKeyIterator) Release() {
	k.i.Next()
}

func (k *levelDBKeyValueStore) Close() {
	k.db.Close()
}

func newLDBKeyValueStore(ldbPath string) (kv kvStore, err error) {
	store := &levelDBKeyValueStore{}
	if store.db, err = leveldb.OpenFile(ldbPath, nil); err != nil {
		return nil, fmt.Errorf("Failed to open DB at %s: %s", ldbPath, err)
	}
	kv = store
	return
}