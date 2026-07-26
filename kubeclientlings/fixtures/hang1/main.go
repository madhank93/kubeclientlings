// hang1 is a fixture for the runner's timeout/cancel tests: it never exits.
// It also spawns nothing, so a plain child kill would suffice — the process
// group behaviour is what `go run` itself adds on top.
package main

import "time"

func main() {
	for {
		time.Sleep(time.Hour)
	}
}
