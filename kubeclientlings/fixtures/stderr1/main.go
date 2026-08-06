// stderr1
// Compiles, runs and succeeds — but writes to stderr on the way out, the way
// client-go's klog does on every real exercise. Verification must judge this a
// pass: the exit code is what decides, not whether stderr stayed empty.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "W0101 00:00:00.000000 1 warning: this is not a failure")
	fmt.Println("ok")
}
