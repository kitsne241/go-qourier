package cprint

import (
	"fmt"
	"log"
)

func CPrintf(format string, args ...any) {
	log.Printf(fmt.Sprintf("\033[220m%s\033[0m", format), args...)
}

func CPanic(format string, args ...any) {
	panic(fmt.Errorf(fmt.Sprintf("\033[204m%s\033[0m", format), args...))
}
