package cprint

import (
	"fmt"
	"log"
)

func CPrintf(format string, args ...any) {
	log.Printf(fmt.Sprintf("\033[204m%s\033[0m", format), args...)
}
