package logging

import (
	"fmt"
	"time"
)

// custom logger
type Logger struct{}

// Write logging output to stdout
func (writer Logger) Write(bytes []byte) (int, error) {
	return fmt.Print(time.Now().UTC().Format(time.StampMicro), " [LOG] ", string(bytes))
}

// Create a new logger
func NewLogger() *Logger {
	return &Logger{}
}
