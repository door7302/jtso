package logger

import (
	"fmt"
	"io"
	"os"
	"runtime/debug"

	"github.com/sirupsen/logrus"
	easy "github.com/t-tomalak/logrus-easy-formatter"
)

var (
	Log     *logrus.Logger
	Verbose bool
	logFile *os.File
)

func StartLogger() {

	f, err := os.OpenFile("/var/log/jtso.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		fmt.Printf("Error opening file: %v", err)
		os.Exit(1)
	}
	logFile = f

	if Verbose {
		Log = &logrus.Logger{
			Out:   io.MultiWriter(os.Stdout, f),
			Level: logrus.DebugLevel,
			Formatter: &easy.Formatter{
				TimestampFormat: "2006-01-02 15:04:05",
				LogFormat:       "%time% [%lvl%] %msg%\n",
			},
		}
	} else {
		Log = &logrus.Logger{
			Out:   f,
			Level: logrus.InfoLevel,
			Formatter: &easy.Formatter{
				TimestampFormat: "2006-01-02 15:04:05",
				LogFormat:       "%time% [%lvl%] %msg%\n",
			},
		}
	}
}

// HandlePanic should be called as: defer logger.HandlePanic()
// It recovers from panics and logs the stack trace.
func HandlePanic() {
	if err := recover(); err != nil {
		Log.Errorf("Recovered from panic: %v", err)
		Log.Error(string(debug.Stack()))
	}
}

// CloseLogger closes the log file handle.
func CloseLogger() {
	if logFile != nil {
		logFile.Close()
	}
}
