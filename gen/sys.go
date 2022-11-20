/*
 * @Date: 2022-11-21 02:19:07
 * @LastEditors: lipengfei
 * @LastEditTime: 2022-11-21 02:21:47
 * @FilePath: \vlgo\gen\sys.go
 * @Description:
 */
package gen

import "vlgo/ecode"

// GenController for control Each GenSvr Start/Stop
type Sys interface {
	Name() string
	PreRun(msg interface{}) bool
	Start(msg interface{}) (interface{}, ecode.VEI)

	PreStop()
	Stop()
}
