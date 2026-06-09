package main

import (
	"sync"
	"sync/atomic"

	"github.com/SmilingXinyi/gb/oss"
)

var (
	handleCounter int64    // atomic counter, starts at 0
	handleStore   sync.Map // int64 → oss.Storage
)

func storeClient(s oss.Storage) int64 {
	id := atomic.AddInt64(&handleCounter, 1)
	handleStore.Store(id, s)
	return id
}

func loadClient(handle int64) (oss.Storage, bool) {
	v, ok := handleStore.Load(handle)
	if !ok {
		return nil, false
	}
	return v.(oss.Storage), true
}

func deleteClient(handle int64) {
	handleStore.Delete(handle)
}
