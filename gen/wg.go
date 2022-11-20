package gen

import (
	"sync"
	"time"

	"vlgo/utils"
)

type WaitReason string

const (
	logSys    = "Sys"
	logWaiter = "Waiter"
)

var AllWaiters sync.Map
var NilWaiter Waiter

func NewWaiter(key string) Waiter {
	if _, ok := AllWaiters.Load(key); ok {
		log.Fatalf(logSys, logWaiter, "create %s already in use", key)
	}

	log.Infof(logSys, logWaiter, "create %s", key)
	ret := Waiter{&sync.WaitGroup{}, key}
	AllWaiters.Store(key, ret)
	return ret
}

// MayWaiter Load Existing Waiter or Store a new waiter for Key
func MayWaiter(key string) Waiter {
	v, loaded := AllWaiters.LoadOrStore(key, Waiter{&sync.WaitGroup{}, key})
	if !loaded {
		log.Infof(logSys, logWaiter, "create %s", key)
	}
	return v.(Waiter)
}

func DelWaiter(key string) {
	if _, loaded := AllWaiters.LoadAndDelete(key); loaded {
		log.Infof(logSys, logWaiter, "delete %s", key)
	}
}

func AddAndSpawnExec(key string, reason any, workerFunc func()) {
	w, ok := FetchWaiter(key)
	if ok {
		w.AddAndSpawnExec(reason, workerFunc)
	} else {
		log.Errorf(logSys, logWaiter, "%v not found for %v", key, reason)
	}
}

func Wait(key string, reason WaitReason) {
	if w, ok := FetchWaiter(key); ok {
		w.Wait(reason)
	}
}

func Done(key string, n int) {
	if w, ok := FetchWaiter(key); ok {
		w.wg.Add(-n)
	}
}

func FetchWaiter(key string) (Waiter, bool) {
	if v, ok := AllWaiters.Load(key); ok {
		return v.(Waiter), ok
	} else {
		return NilWaiter, false
	}
}


// Waiter wrapper for sync.WaitGroup with Key
type Waiter struct {
	wg  *sync.WaitGroup // 不做匿名，防止外部调用
	Key string
}

// AddAndSpawnExec add one counter on wg and start one goroutine exec mainFunc logic
func (w Waiter) AddAndSpawnExec(reason any, mainFunc func()) {
	if w.wg == nil {
		log.Errorf(logSys, logWaiter, "waiter not init")
		return
	}

	w.wg.Add(1)
	log.Infof(logSys, logWaiter, "[%s] add count %v for %v", w.Key, 1, reason)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf(logSys, logWaiter, "[%s]:[%v] panic: %v, stack: %s", w.Key, reason, err, utils.Stack())
			}
			w.done(reason)
		}()

		mainFunc()
	}()
}

func (w Waiter) Add(n int, reason any) {
	if w.wg != nil {
		log.Infof(logSys, logWaiter, "[%s] add count %v for %v", w.Key, n, reason)
		w.wg.Add(n)
	} else {
		log.Errorf(logSys, logWaiter, "waiter not init")
	}
}

func (w Waiter) Run(reason any, workerFunc func()) {
	go func() {
		defer func() {
			if err := recover(); err != nil {
				log.Errorf(logSys, logWaiter, "[%s]:[%v] panic: %v, stack: %s", w.Key, reason, err, utils.Stack())
			}
			w.done(reason)
		}()
		workerFunc()
	}()
}

func (w Waiter) DoneOne(reason any) {
	w.done(reason)
}

func (w Waiter) WaitInfinity(reason WaitReason) {
	if w.wg != nil {
		log.Infof(logSys, logWaiter, "[%s] wait infinity for %v", w.Key, reason)
		w.wg.Wait()
		log.Infof(logSys, logWaiter, "[%s] wait infinity returned %v", w.Key, reason)
	} else {
		log.Errorf(logSys, logWaiter, "waiter not init")
	}
}

func (w Waiter) Wait(reason WaitReason) {
	if w.wg != nil {
		log.Infof(logSys, logWaiter, "[%s] wait for %v", w.Key, reason)

		waitChan := make(chan struct{}, 0)
		go func() {
			w.wg.Wait()
			waitChan <- struct{}{}
		}()
		overTimer := time.NewTimer(time.Second * 7)
		select {
		case <-waitChan:
		case <-overTimer.C:
		}
		log.Infof(logSys, logWaiter, "[%s] wait returned %v", w.Key, reason)
	} else {
		log.Errorf(logSys, logWaiter, "waiter not init")
	}
}

func (w Waiter) done(reason any) {
	if w.wg != nil {
		log.Infof(logSys, logWaiter, "[%s] done for %v", w.Key, reason)
		w.wg.Done()
	} else {
		log.Errorf(logSys, logWaiter, "waiter not init")
	}
}
