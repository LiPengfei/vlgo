/*
 * @Date: 2022-11-21 00:42:22
 * @LastEditors: Please set LastEditors
 * @LastEditTime: 2022-11-21 09:45:53
 * @FilePath: /vlgo/logger/logger.go
 * @Description:
 */
package logger

import (
	"io"
	"os"
	"path/filepath"
	"time"
	"vlgo/logger/encoder"
	"vlgo/logger/logproxy"

	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

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

type DebugEnabled struct{}
type InfoEnabled struct{}
type ErrorEnabled struct{}

func (d DebugEnabled) Enabled(level zapcore.Level) bool {
	return level >= zapcore.DebugLevel
}

func (l InfoEnabled) Enabled(level zapcore.Level) bool {
	return level >= zapcore.InfoLevel
}

func (e ErrorEnabled) Enabled(level zapcore.Level) bool {
	return level >= zapcore.ErrorLevel
}

type InitParam struct {
	LogLevel     string
	LogFileSize  int64
	LogFileCount uint
}

func initZapLogger(pathName string, logName string, param *InitParam, logLvAtom zap.AtomicLevel) (*zap.Logger, error) {
	infoCore, err := getInfoCore(pathName, logName, param)
	if err != nil {
		return nil, err
	}

	errorCore, err := getErrorCore(pathName, logName, param)
	if err != nil {
		return nil, err
	}

	consoleCore, err := getConsoleCore(true, param.LogLevel, logLvAtom)
	if err != nil {
		return nil, err
	}

	if param.LogLevel == "debug" {
		debugCore, err := getDebugCore(pathName, logName, param, logLvAtom)
		if err != nil {
			return nil, err
		}
		return zap.New(zapcore.NewTee(infoCore, errorCore, debugCore, consoleCore), zap.AddCaller(), zap.AddCallerSkip(1)), nil
	}
	return zap.New(zapcore.NewTee(infoCore, errorCore, consoleCore), zap.AddCaller(), zap.AddCallerSkip(1)), nil
}

func getInfoCore(pathName string, logName string, param *InitParam) (zapcore.Core, error) {
	logEncoder := zap.NewProductionEncoderConfig()
	logEncoder.EncodeTime = zapcore.ISO8601TimeEncoder

	infoWriter := getLoggerWriter(pathName, logName+"_info", param)
	writeSyncer := zapcore.AddSync(infoWriter)

	return zapcore.NewCore(zapcore.NewJSONEncoder(logEncoder), writeSyncer, InfoEnabled{}), nil
}

func getErrorCore(pathName string, logName string, param *InitParam) (zapcore.Core, error) {
	logEncoder := zap.NewProductionEncoderConfig()
	logEncoder.EncodeTime = zapcore.ISO8601TimeEncoder

	infoWriter := getLoggerWriter(pathName, logName+"_error", param)
	writeSyncer := zapcore.AddSync(infoWriter)

	return zapcore.NewCore(zapcore.NewJSONEncoder(logEncoder), writeSyncer, ErrorEnabled{}), nil
}

func getDebugCore(pathName string, logName string, param *InitParam, atomLogLevel zap.AtomicLevel) (zapcore.Core, error) {
	logEncoder := zap.NewProductionEncoderConfig()
	logEncoder.EncodeTime = zapcore.ISO8601TimeEncoder

	infoWriter := getLoggerWriter(pathName, logName+"_debug", param)
	writeSyncer := zapcore.AddSync(infoWriter)

	return zapcore.NewCore(zapcore.NewJSONEncoder(logEncoder), writeSyncer, atomLogLevel), nil
}

func getConsoleCore(colorLevel bool, level string, atomLogLevel zap.AtomicLevel) (zapcore.Core, error) {
	consoleEncoder := zap.NewProductionEncoderConfig()
	consoleEncoder.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString("[")
		enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
		enc.AppendString("]")
	}
	consoleEncoder.EncodeLevel = encoder.MyLevelEncoder
	consoleEncoder.EncodeCaller = encoder.MyCallerEncode

	if colorLevel {
		consoleEncoder.EncodeLevel = encoder.MyColorLevelEncoder
	}

	writeSyncer := zapcore.AddSync(logproxy.NewLogProxyWriter(os.Stdout))
	return zapcore.NewCore(encoder.NewConsoleEncoder(consoleEncoder), writeSyncer, atomLogLevel), nil
}

// 日志分割
func getLoggerWriter(pathname string, filename string, param *InitParam) io.Writer {
	// 保存7份的日志, 一天滚动一次
	// 根据内部实现，写入日志时，判断文件名是否更新
	// 如果文件名更新，则滚动
	// 文件名是根据时间来的，这个时间是取余计算, 然后把余数抹掉计算出来的，
	// 如果取的是86400s的余，其实每天是0点能够准时刷新的
	hook, err := rotatelogs.New(
		filepath.Join(pathname, filename+"_%Y-%m-%d"+".log"),
		rotatelogs.WithRotationCount(7),
		rotatelogs.WithRotationTime(time.Hour*24),
		//日志文件数量限制有bug，不是按照顺序删除旧日志，暂时不用此功能
		//rotatelogs.WithRotationSize(param.LogFileSize),
	)
	if err != nil {
		panic(err)
	}
	return logproxy.NewLogProxyWriter(hook)
}
