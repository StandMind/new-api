package controller

import (
	"sync"

	"github.com/QuantumNous/new-api/common"
)

var disableControllerTestRedis sync.Once

func disableRedisForControllerTests() {
	disableControllerTestRedis.Do(func() {
		common.RedisEnabled = false
	})
}
