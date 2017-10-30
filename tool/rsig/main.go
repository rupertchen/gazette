package main

import (
	"github.com/LiveRamp/gazette/pprof"

	"fmt"

	"github.com/sirupsen/logrus"
)

func main() {
	pprof.ToggleProfiler()

	for i := 0; i < 44; i++ {
		fmt.Println(fib(i))
	}

	pprof.ToggleProfiler()
}

func fib(n int) int {
	if n < 0 {
		logrus.Fatal("Dead. Can't do negative fib!")
	}
	switch n {
	case 0, 1:
		return 1
	default:
		return fib(n-1) + fib(n-2)
	}
}
