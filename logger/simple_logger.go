/*
 * @Author: lipengfei
 * @Date: 2022-11-21 09:44:03
 * @LastEditTime: 2022-11-21 09:52:03
 * @FilePath: /vlgo/logger/simple_logger.go
 * @Description:
 */
package logger

import (
	"fmt"

	"github.com/petermattis/goid"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type SimpleLogger struct {
	Dir          string
	Logger       *zap.Logger
	AtomLogLevel zap.AtomicLevel
}

var SLog *SimpleLogger

func (l *SimpleLogger) Debugf(sys, tag, fmts string, infos ...interface{}) {
	if l.Logger.Core().Enabled(zap.DebugLevel) {
		l.Logger.Debug(sys+"#"+tag, zap.Stringer("m", str(fmts, infos...)))
	}
}

func (l *SimpleLogger) Infof(sys, tag, fmts string, infos ...interface{}) {
	l.Logger.Info(sys+"#"+tag, zap.Stringer("m", str(fmts, infos...)))
}

func (l *SimpleLogger) Warnf(sys, tag, fmts string, infos ...interface{}) {
	l.Logger.Warn(sys+"#"+tag, zap.Stringer("m", str(fmts, infos...)))
}

func (l *SimpleLogger) Errorf(sys, tag, fmts string, infos ...interface{}) {
	l.Logger.Error(sys+"#"+tag, zap.Stringer("m", str(fmts, infos...)))
}

func (l *SimpleLogger) Panicf(sys, tag, fmts string, infos ...interface{}) {
	l.Logger.Panic(sys+"#"+tag, zap.Stringer("m", str(fmts, infos...)))
}

func (l *SimpleLogger) Fatalf(sys, tag, fmts string, infos ...interface{}) {
	l.Logger.Fatal(sys+"#"+tag, zap.Stringer("m", str(fmts, infos...)))
}

func InitSimpleLog(fn string, param *InitParam) error {
	retLog, err := initLog(fn, param)
	if err != nil {
		return err
	}
	SLog = retLog
	return nil
}

func ReloadLogLv(cfg *viper.Viper) {
	oldLv := SLog.AtomLogLevel.Level()

	strLevel := cfg.GetString("logging.level")
	var lv zapcore.Level
	err := lv.Set(strLevel)
	if err != nil {
		return
	}
	if oldLv == lv {
		return
	}
	SLog.AtomLogLevel.SetLevel(lv)
}

type logStringer struct {
	goID  int64
	fmts  string
	infos []interface{}
}

func (v *logStringer) String() string {
	return fmt.Sprintf("[Go #%v] ", v.goID) + fmt.Sprintf(v.fmts, v.infos...)
}

func str(fmts string, infos ...interface{}) *logStringer {
	return &logStringer{
		goID:  goid.Get(),
		fmts:  fmts,
		infos: infos,
	}
}

func initLog(fn string, param *InitParam) (*SimpleLogger, error) {
	var lv zapcore.Level
	err := lv.Set(param.LogLevel)
	if err != nil {
		return nil, err
	}

	ret := &SimpleLogger{
		Dir:          "logs",
		AtomLogLevel: zap.NewAtomicLevel(),
	}
	ret.AtomLogLevel.SetLevel(lv)

	if l, err := initZapLogger(ret.Dir, fn, param, ret.AtomLogLevel); err != nil {
		ret.Logger = l
	}
	return ret, nil
}
