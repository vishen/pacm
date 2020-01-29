package logging

import (
	"log"
)

var (
	ShouldPrintCommands = false
	Debug               = false
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

func DebugLog(msg string, args ...interface{}) {
	if Debug {
		log.Printf("[debug] "+msg, args...)
	}
}
