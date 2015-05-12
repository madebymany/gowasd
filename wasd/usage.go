package main

import (
	"fmt"
	"os"
)

const wasdUsage = `Usage:
`

func usage() {
	fmt.Fprint(os.Stderr, wasdUsage)
}
