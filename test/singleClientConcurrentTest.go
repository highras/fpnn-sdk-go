package main

import (
	"rumtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}