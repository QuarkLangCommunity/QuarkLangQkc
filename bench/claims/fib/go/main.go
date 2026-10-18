package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

func fib(n int) int {
	if n < 2 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func main() {
	n := 35
	if len(os.Args) > 1 {
		if v, err := strconv.Atoi(os.Args[1]); err == nil {
			n = v
		}
	}
	t0 := time.Now()
	r := fib(n)
	us := time.Since(t0).Microseconds()
	fmt.Println(us)
	fmt.Println(r)
}
