package main

import (
	"fmt"

	"github.com/nkbai/tgo/lib/tracer"
)

//go:noinline
func tracedFunc() []int {
	fmt.Println("traced")
	return nil
}

func main() {
	tracer.SetVerboseOption(true)
	tracer.SetTraceLevel(2)
	if err := tracer.Start(); err != nil {
		panic(err)
	}

	fmt.Println("traced")

	// start again (should be no-op)
	if err := tracer.Start(); err != nil {
		panic(err)
	}

	arr := tracedFunc()
	arr = append(arr, 1)

	tracer.Stop()

	fmt.Println("not traced", arr)

	// start again
	if err := tracer.Start(); err != nil {
		panic(err)
	}
}
