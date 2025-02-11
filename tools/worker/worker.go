package worker

import (
	"fmt"
	"os"
)

func RunWorker() {
	fmt.Fprintf(os.Stderr, "Starting worker mode with args: %v\n", os.Args)
}
