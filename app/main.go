package main

import (
	"fmt"
	"os"
	"bufio"
	"strings" 
)

// Ensures gofmt doesn't remove the "fmt" and "os" imports in stage 1 (feel free to remove this!)
var _ = fmt.Fprint
var _ = os.Stdout

func main() {
	// TODO: Uncomment the code below to pass the first stage
	reader := bufio.Reader(os.Stdin)
	line, err := reader.ReadString('\n')
	fmt.Fprint(os.Stdout, "$ ")

	if err != nil {
		return
	}

	command := strings.TrimSpace(line)
	fmt.Printf(os.Stdout, "%s: command not found\n", command)
}
