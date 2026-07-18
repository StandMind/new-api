package common

import "sync/atomic"

var processReady atomic.Bool

func SetProcessReady(ready bool) {
	processReady.Store(ready)
}

func IsProcessReady() bool {
	return processReady.Load()
}
