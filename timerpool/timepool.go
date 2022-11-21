/*
 * @Author: lipengfei
 * @Date: 2022-11-21 08:04:20
 * @LastEditTime: 2022-11-21 08:04:22
 * @FilePath: /vlgo/timerpool/timepool.go
 * @Description:
 */
package timerpool

import (
	"sync"
	"time"
)

var globalTimerPool = sync.Pool{}

// GetTimer timer需要在同一个协程接收超时及回收，否则将会出现并发问题
func GetTimer(d time.Duration) *time.Timer {
	if t, _ := globalTimerPool.Get().(*time.Timer); t != nil {
		t.Reset(d)
		return t
	}

	return time.NewTimer(d)
}

func PutTimer(t *time.Timer) {
	if !t.Stop() {
		select {
		case <-t.C:
		default:
		}
	}
	globalTimerPool.Put(t)
}
