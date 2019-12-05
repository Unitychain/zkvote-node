package utils

import (
	"fmt"
	"path/filepath"

	"os"
	"runtime"
	"strings"
	"strconv"

	logging "github.com/op/go-logging"
)

var logger = logging.MustGetLogger("logs")

// var formatConsole = logging.MustStringFormatter(
// 	`%{color}%{time:15:04:05.00} [%{level:.5s}] %{color:reset}%{message}`, // (%{shortfile}
// )
var formatConsole = logging.MustStringFormatter(
	`%{color}%{time:15:04:05.000} %{color:reset}%{message}`, // (%{shortfile}
)

var formatFile = logging.MustStringFormatter(
	`%{color}%{time:2006-01-02 15:04:05.000} [%{level:.5s}] %{color:reset}%{message}`, // (%{shortfunc}___%{shortfile}
)

var file *os.File = nil

// OpenLog ...
func OpenLog() {
	if file != nil {
		return
	}

	var err error
	file, err = os.OpenFile("logs.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
	if err != nil {
		fmt.Printf("error opening file: %v", err)
	}
	//defer f.Close()

	backendFile := logging.NewLogBackend(file, "", 0)
	backendFileFormatter := logging.NewBackendFormatter(backendFile, formatFile)

	backendConsole := logging.NewLogBackend(os.Stdout, "", 0)
	backendConsoleFormatter := logging.NewBackendFormatter(backendConsole, formatConsole)
	backendConsoleLeveled := logging.AddModuleLevel(backendConsoleFormatter)
	//backendConsoleLeveled.SetLevel(logging.WARNING, "")

	logging.SetBackend(backendFileFormatter, backendConsoleLeveled)

}

// CloseLog ...
func CloseLog() {
	if file != nil {
		file.Close()
	}
}

// AssertError ...
func AssertError(err error, msg string) bool {
	if err == nil {
		return false
	}
	LogErrorf("%s - [%s]", msg, err.Error())
	return true
}

// AssertWarn ...
func AssertWarn(err error, msg string) bool {
	if err == nil {
		return false
	}
	LogWarningf("%s - [%s]", msg, err.Error())
	return true
}

func getFileNLine() string {
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "???"
		line = 0
	}
	return fmt.Sprintf("%s:%d", filepath.Base(file), line)
}

// LogDebugf ...
func LogDebugf(s string, args ...interface{}) {
	logger.Debugf("%s", fmt.Sprintf(s, args...))
}

// LogDebug ...
func LogDebug(s string) {
	logger.Debug(s)
}

// LogInfof ...
func LogInfof(s string, args ...interface{}) {
	logger.Infof("[%d] %s (%s)", Goid(), fmt.Sprintf(s, args...), getFileNLine())
}

// LogInfo ...
func LogInfo(s string) {

	logger.Infof("[%d] %s (%s)", Goid(), s, getFileNLine())
}

// LogWarningf ...
func LogWarningf(s string, args ...interface{}) {
	logger.Warningf("[%d] %s (%s)", Goid(), fmt.Sprintf(s, args...), getFileNLine())
}

// LogWarning ...
func LogWarning(s string) {
	logger.Warningf("[%d] %s (%s)", Goid(), s, getFileNLine())
}

// LogErrorf ...
func LogErrorf(s string, args ...interface{}) {
	logger.Errorf("[%d] %s (%s)", Goid(), fmt.Sprintf(s, args...), getFileNLine())
}

// LogError ...
func LogError(s string) {
	logger.Errorf("[%d] %s (%s)", Goid(), s, getFileNLine())
}

// LogFatalf ...
func LogFatalf(s string, args ...interface{}) {
	logger.Fatalf("[%d] %s (%s)", Goid(), fmt.Sprintf(s, args...), getFileNLine())
}

// LogFatal ...
func LogFatal(s string) {
	logger.Fatalf("[%d] %s (%s)", Goid(), s, getFileNLine())
}

func Goid() int {
	defer func() {
		if err := recover(); err != nil {
			logger.Fatal("panic recover:panic info:%v", err)
		}
	}()

	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.Atoi(idField)
	if err != nil {
		panic(fmt.Sprintf("cannot get goroutine id: %v", err))
	}
	return id
}