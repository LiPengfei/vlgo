/*
 * @Date: 2022-11-21 01:53:54
 * @LastEditors: lipengfei
 * @LastEditTime: 2022-11-21 01:53:59
 * @FilePath: \vlgo\gen\gen.go
 * @Description:
 */
package gen

import "vlgo/logger"

var log logger.Logger

func InitLog(l logger.Logger) {
	log = l
}
