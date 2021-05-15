package printpretty

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// #include "Windows.h"
var (
	reset  = "\033[0m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	gray   = "\033[90m"
	white  = "\033[97m"
)

func init() {
	if runtime.GOOS == "windows" {
		reset = ""
		red = ""
		green = ""
		yellow = ""
		gray = ""
		white = ""
	}
}

type messageType int

// Enum for status levels
const (
	INFO messageType = iota
	WARNING
	ERROR
	FATAL
	QUIET
)

func printPretty(messageType messageType, message string, args ...interface{}) {
	color := white

	formattedMessage := fmt.Sprintf(message, args...)

	switch messageType {
	case QUIET:
		color = gray
	case WARNING:
		color = yellow
	case ERROR:
		color = red
	}

	fmt.Printf("[%s] %s\r\n", time.Now().Local().Format("15:04:05.000"), sprintc(color, formattedMessage))
}

func sprintc(color, str string) string {
	return color + str + reset
}

// Quiet prints a message with gray text
func Quiet(message string, args ...interface{}) {
	printPretty(QUIET, message, args...)
}

// Info prints a message with white text
func Info(message string, args ...interface{}) {
	printPretty(INFO, message, args...)
}

// Warn prints a message with yellow text
func Warn(message string, args ...interface{}) {
	printPretty(WARNING, message, args...)
}

// Error prints a message with red text
func Error(message string, args ...interface{}) {
	printPretty(ERROR, message, args...)
}

// Highlight searches for a substring and highlights it green
func Highlight(message, command string, args ...interface{}) {
	formattedMessage := strings.ReplaceAll(message, command, green+command+reset)
	printPretty(INFO, formattedMessage, args...)
}
