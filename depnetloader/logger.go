package depnetloader

import (
	"fmt"
	"os"
)

var EnableDebug = false

func debugf(format string, args ...interface{}) {
	if EnableDebug {
		fmt.Fprintln(os.Stderr, fmt.Sprintf(format, args...))
	}
}
