package common

import (
	"encoding/json"
	"fmt"
	"io"

	"camel.uangel.com/ua5g/ulib.git/ulog"
	"github.com/labstack/gommon/log"
)

// ULogWriter ULOG Writer
type ULogWriter struct {
	ulog ulog.Logger
}

// EchoLogger Echo Server용 logger
type EchoLogger struct {
	out    *ULogWriter
	level  log.Lvl
	prefix string
}

const httplogname = "com.uangel.unrf.http"

// logger http loggger
var httplogger = ulog.GetLogger(httplogname)

// Default Echo logger
var defaulEchoLogger = NewEchoLogger(httplogname)

////////////////////////////////////////////////////////////////////////////////
// ULogWriter functions
////////////////////////////////////////////////////////////////////////////////

// Write ULog Writer
func (w *ULogWriter) Write(p []byte) (n int, err error) {
	w.ulog.Print(string(p))
	return len(p), nil
}

////////////////////////////////////////////////////////////////////////////////
// EchoLogger functions
////////////////////////////////////////////////////////////////////////////////

//DefaultEchoLogger return singleton logger
func DefaultEchoLogger() *EchoLogger {
	return defaulEchoLogger
}

// NewEchoLogger 새로운 Echo Logger를 생성한다.
func NewEchoLogger(name string) *EchoLogger {
	l := &EchoLogger{}
	l.out = &ULogWriter{ulog: ulog.GetLogger(name)}
	l.level = log.OFF
	l.prefix = "http"

	return l
}

// Output return logger io.Writer
func (l *EchoLogger) Output() io.Writer {
	return l.out
}

// SetOutput logger io.Writer
func (l *EchoLogger) SetOutput(w io.Writer) {
	//Do Nothing
}

// Level return logger level
func (l *EchoLogger) Level() log.Lvl {
	return l.level
}

// SetLevel logger level
func (l *EchoLogger) SetLevel(v log.Lvl) {
	l.level = v
}

// Prefix return logger prefix
func (l *EchoLogger) Prefix() string {
	return l.prefix
}

// SetPrefix logger prefix
func (l *EchoLogger) SetPrefix(p string) {
	l.prefix = p
}

// Print output message of print level
func (l *EchoLogger) Print(i ...interface{}) {
	l.out.ulog.Print(i...)
}

// Printf output format message of print level
func (l *EchoLogger) Printf(format string, args ...interface{}) {
	l.out.ulog.Print(fmt.Sprintf(format, args...))
}

// Printj output json of print level
func (l *EchoLogger) Printj(j log.JSON) {
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	l.out.ulog.Print(string(b))
}

// Debug output message of debug level
func (l *EchoLogger) Debug(i ...interface{}) {
	if log.DEBUG < l.level {
		return
	}
	l.out.ulog.Debug(fmt.Sprint(i...))
}

// Debugf output format message of debug level
func (l *EchoLogger) Debugf(format string, args ...interface{}) {
	if log.DEBUG < l.level {
		return
	}
	l.out.ulog.Debug(format, args...)
}

// Debugj output message of debug level
func (l *EchoLogger) Debugj(j log.JSON) {
	if log.DEBUG < l.level {
		return
	}
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	l.out.ulog.Debug(string(b))
}

// Info output message of info level
func (l *EchoLogger) Info(i ...interface{}) {
	if log.INFO < l.level {
		return
	}
	l.out.ulog.Info(fmt.Sprint(i...))
}

// Infof output format message of info level
func (l *EchoLogger) Infof(format string, args ...interface{}) {
	if log.INFO < l.level {
		return
	}
	l.out.ulog.Info(format, args...)
}

// Infoj output json of info level
func (l *EchoLogger) Infoj(j log.JSON) {
	if log.INFO < l.level {
		return
	}
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	l.out.ulog.Info(string(b))
}

// Warn output message of warn level
func (l *EchoLogger) Warn(i ...interface{}) {
	if log.WARN < l.level {
		return
	}
	l.out.ulog.Warn(fmt.Sprint(i...))
}

// Warnf output format message of warn level
func (l *EchoLogger) Warnf(format string, args ...interface{}) {
	if log.WARN < l.level {
		return
	}
	l.out.ulog.Warn(format, args...)
}

// Warnj output json of warn level
func (l *EchoLogger) Warnj(j log.JSON) {
	if log.WARN < l.level {
		return
	}
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	l.out.ulog.Warn(string(b))
}

// Error output message of error level
func (l *EchoLogger) Error(i ...interface{}) {
	if log.ERROR < l.level {
		return
	}
	l.out.ulog.Warn(fmt.Sprint(i...))
}

// Errorf output format message of error level
func (l *EchoLogger) Errorf(format string, args ...interface{}) {
	if log.ERROR < l.level {
		return
	}
	l.out.ulog.Error(format, args...)
}

// Errorj output json of error level
func (l *EchoLogger) Errorj(j log.JSON) {
	if log.ERROR < l.level {
		return
	}
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	l.out.ulog.Error(string(b))
}

// Fatal output message of fatal level
func (l *EchoLogger) Fatal(i ...interface{}) {
	l.out.ulog.Fatal(fmt.Sprint(i...))
}

// Fatalf output format message of fatal level
func (l *EchoLogger) Fatalf(format string, args ...interface{}) {
	l.out.ulog.Fatal(format, args...)
}

// Fatalj output json of fatal level
func (l *EchoLogger) Fatalj(j log.JSON) {
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	l.out.ulog.Fatal(string(b))
}

// Panic output message of panic level
func (l *EchoLogger) Panic(i ...interface{}) {
	l.out.ulog.Panic(fmt.Sprint(i...))
}

// Panicf output format message of panic level
func (l *EchoLogger) Panicf(format string, args ...interface{}) {
	l.out.ulog.Panic(format, args...)
}

// Panicj output json of panic level
func (l *EchoLogger) Panicj(j log.JSON) {
	b, err := json.Marshal(j)
	if err != nil {
		panic(err)
	}
	l.out.ulog.Panic(string(b))
}

func (l *EchoLogger) SetHeader(h string) {

}
