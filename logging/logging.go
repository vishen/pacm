package logging

import "log"

func PrintCommand(msg string, args ...interface{}) {
	log.Printf(msg, args...)
}
