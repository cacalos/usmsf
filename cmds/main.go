package main

import (
	"fmt"
	"os"
)

// Myprint print hello
func Myprint() {
	fmt.Fprintf(os.Stdout, "%v\n", "Hello")
}
func main() {
	Myprint()
	os.Exit(0)
}
