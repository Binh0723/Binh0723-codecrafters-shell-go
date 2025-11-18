package main

import (
	"fmt"
	"os"
	"bufio"
)

// Ensures gofmt doesn't remove the "fmt" and "os" imports in stage 1 (feel free to remove this!)
var _ = fmt.Fprint
var _ = os.Stdout

func main() {
	// TODO: Uncomment the code below to pass the first stage
	reader := bufio.Reader(os.Stdin)
	line, err := reader.ReadString('\n')
	fmt.Fprintf(os.Stdout, "%s: command not found \n", line)
}
