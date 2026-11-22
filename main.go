package main

import (
	"fmt"
	"middleman/program"
	"os"
)

func main() {
	p := program.NewProgram()
	output, code := p.Main(os.Args[1:])
	if output != "" {
		fmt.Println(output)
	}
	os.Exit(code)
}
