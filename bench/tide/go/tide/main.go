// Tide control, observable allocation: the same three-int allocate/use/drop loop as the QuarkLang
// benchmark. The global sink forces the slice to the heap; resetting it each round keeps the work
// bounded while keeping every allocation observable.
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

var sink []int32

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
		sink = v
		acc += int64(v[0] + v[2])
		sink = nil
	}
	fmt.Println(time.Since(t0).Microseconds())
	fmt.Println(acc)
}
