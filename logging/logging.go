package logging

import (
	"log"
)

var (
	ShouldPrintCommands = false
)

func PrintCommand(msg string, args ...interface{}) {
	if !ShouldPrintCommands {
		return
	}
	log.Printf(msg, args...)
}

func ErrorLog(msg string, args ...interface{}) {
	log.Printf("[error] "+msg, args...)
}
