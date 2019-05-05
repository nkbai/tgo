package main

import (
	"fmt"

	"github.com/nkbai/tgo/lib/tracer"
)

func fib(n int) int {
	if n == 0 || n == 1 {
		return n
	}
	return fib(n-1) + fib(n-2)
}

func main() {
	tracer.SetTraceLevel(2)
	tracer.Start()

	val := fib(3)
	fmt.Println(val)

	tracer.Stop()
}
