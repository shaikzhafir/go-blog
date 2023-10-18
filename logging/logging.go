package logging

import (
	"fmt"
	"os"

	"log/slog"
)

var logger *slog.Logger

func init() {
	var file *os.File
	logDir := "/opt/blog/blog.log"
	if os.Getenv("DEV") != "true" {
		var err error
		file, err = os.OpenFile(logDir, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			panic(err)
		}
	} else {
		file = os.Stdout
	}
	// Set up the logger with file as the output destination
	logger = slog.New(slog.NewJSONHandler(file, nil))
}

// Info logs informational messages
func Info(format string, a ...interface{}) {
	message := fmt.Sprintf(format, a...)
	logger.Info(message)
}

func Error(format string, a ...interface{}) {
	logger.Error(format, a...)
}

func Fatal(format string, a ...interface{}) {
	logger.Error(format, a...)
	os.Exit(1)
}
