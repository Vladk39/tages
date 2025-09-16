package helper

import (
	"fmt"
	"time"
)

func UniqueFilename(filename string) string {
	now := time.Now()
	timestamp := now.Format("20060102_150405")
	nanos := now.Nanosecond()
	return fmt.Sprintf("%s_%d_%s", timestamp, nanos, filename)
}
