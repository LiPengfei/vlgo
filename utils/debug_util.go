/*
 * @Date: 2022-11-21 02:01:08
 * @LastEditors: lipengfei
 * @LastEditTime: 2022-11-21 02:16:16
 * @FilePath: \vlgo\utils\debug_util.go
 * @Description:
 */

package utils

import (
	"os"
	"os/user"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

const (
	OmitStackHeadLen = 7
	RoutineDelta     = 10
	PIDBarrier       = 10

	FlushGONumTick = time.Second * 60
	ForceGCTick    = time.Minute * 10
	MemStatTick    = time.Minute * 11

	OutputDir   = "./logs"
	GoNumFile   = OutputDir + "/go.nums"
	MemStatFile = OutputDir + "/memstat.log"
)

var IsInDocker = false

// SizedStack Stack use a max 4096 bytes array to hold stack
func SizedStack() string {
	var buf [4 << 10]byte
	return string(buf[:runtime.Stack(buf[:], false)])
}

// Stack return a large enough buf to hold the call stack
// log里面 fmt用 %s, 会自动换行
func Stack() string {
	rawStack := debug.Stack()
	s := string(rawStack)
	lines := strings.Split(s, "\n")
	if len(lines) >= OmitStackHeadLen {
		// fmt.Println("orig stack: ", s)
		newLines := append(lines[0:0], lines[OmitStackHeadLen:]...)
		return strings.Join(newLines, "\n")
	}
	return s
}

func ForceGC() {
	runtime.GC()
}

// ForceReturnMem call FreeOSMemory forces a garbage collection followed by an attempt to return as much memory
// to the operating system as possible. (Even if this is not called, the runtime gradually returns memory
// to the operating system in a background task.)
func ForceReturnMem() {
	debug.FreeOSMemory()
}

func PID() int {
	return os.Getpid()
}

func CurrentUser() string {
	current, err := user.Current()
	if err != nil {
		return ""
	}
	return current.Username
}

func NumGoroutine() int {
	return runtime.NumGoroutine()
}

func PIDFileName(appName string) string {
	return appName + ".pid"
}
