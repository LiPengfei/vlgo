/*
 * @Date: 2022-11-21 00:42:22
 * @LastEditors: lipengfei
 * @LastEditTime: 2022-11-21 01:55:25
 * @FilePath: \vlgo\logger\logger.go
 * @Description:
 */
package logger

type Logger interface {
	// Debugf logs a message at DebugLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	Debugf(sys, tag, fmts string, infos ...interface{})

	// Infof logs a message at InfoLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	Infof(sys, tag, fmts string, infos ...interface{})

	// Warnf logs a message at WarnLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	Warnf(sys, tag, fmts string, infos ...interface{})

	Errorf(sys, tag, fmts string, infos ...interface{})

	// Panicf logs a message at PanicLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	//
	// The logger then panics, even if logging at PanicLevel is disabled.
	Panicf(sys, tag, fmts string, infos ...interface{})

	// Fatalf logs a message at FatalLevel. The message includes any fields passed
	// at the log site, as well as any fields accumulated on the logger.
	//
	// The logger then calls os.Exit(1), even if logging at FatalLevel is
	// disabled.
	Fatalf(sys, tag, fmts string, infos ...interface{})
}
