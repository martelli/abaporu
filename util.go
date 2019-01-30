package main

import (
	"fmt"
	"time"
)

func ReadTillNull(buf []byte) ([]byte, []byte, error) {
	for i := 0; i < len(buf); i++ {
		if buf[i] == 0 {
			return buf[:i], buf[i+1:], nil
		}
	}
	return nil, nil, fmt.Errorf("Could not find null terminator")
}

func TimeOnly() string {
	return time.Now().Format("15:04:05.000")
}
