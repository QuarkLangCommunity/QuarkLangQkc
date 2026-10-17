// Tide control, dead-code floor: the same loop with a local slice and no sink, so the Go compiler
// may keep the slice on the stack. The checksum keeps the loop itself alive.
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// main runs the control loop and prints elapsed microseconds and the checksum.
func main() {
	n := int64(100_000_000)
	if len(os.Args) > 1 {
		if v, err := strconv.ParseInt(os.Args[1], 10, 64); err == nil {
			n = v
		}
	}
	var acc int64
	t0 := time.Now()
	for i := int64(0); i < n; i++ {
		v := make([]int32, 3)
		v[0] = int32(i % 7)
		v[1] = int32(i % 11)
		v[2] = int32(i % 13)
		acc += int64(v[0] + v[2])
	}
	fmt.Println(time.Since(t0).Microseconds())
	fmt.Println(acc)
}
