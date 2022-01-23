package main

import (
	"fake-proxy/program"
	"os"
)

func main() {
	program.Main(os.Args[1:])
}
