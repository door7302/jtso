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
)

func StartLogger() {

	f, err := os.OpenFile("/var/log/jtso_enricher.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)

	if err != nil {
		fmt.Printf("Error opening file: %v", err)
		os.Exit(0)
	}

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

// Redirect panic in logger file
func HandlePanic() {
	defer func() {
		if err := recover(); err != nil {
			Log.Error(string(debug.Stack()))
		}
	}()
}
