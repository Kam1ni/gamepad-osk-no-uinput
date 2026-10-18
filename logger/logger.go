package logger

import (
	"fmt"
	"log"
	"os"
)

var outLogger = log.New(os.Stdout, "", 0)
var errLogger = log.New(os.Stderr, "", 0)

func Out(pattern string, args ...interface{}) {
	message := fmt.Sprintf(pattern, args...)
	outLogger.Println(message)
}

func Err(pattern string, args ...interface{}) {
	message := fmt.Sprintf(pattern, args...)
	errLogger.Println(message)
}
