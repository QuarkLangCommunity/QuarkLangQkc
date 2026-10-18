package main

import (
	"fmt"
	"time"
)

// main sums the same 1e9 terms with a plain loop and prints the elapsed microseconds and the int32
// checksum the QuarkLang closed form also produces.
func main() {
	const n = 1000000000
	t0 := time.Now()
	var s int64
	for i := int64(0); i < n; i++ {
		s += i*3 + 1
	}
	r := int32(s)
	us := time.Since(t0).Microseconds()
	fmt.Println(us)
	fmt.Println(r)
}
