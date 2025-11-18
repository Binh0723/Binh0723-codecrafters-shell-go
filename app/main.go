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
	for {
		fmt.Fprint(os.Stdout, "$ ")
		reader := bufio.NewReader(os.Stdin)
		line, err := reader.ReadString('\n')

		if err != nil {
			return
		}

		command := strings.TrimSpace(line)
		parts := strings.Split(command, " ")
		if parts[0] == "exit" {
			return
		} else if parts[0] == "echo" {
			new_line := strings.Join(parts[1:], " ")
			fmt.Fprintf(os.Stdout, "%s\n", new_line)
		} else {
			fmt.Fprintf(os.Stdout, "%s: command not found\n", command)
		}	
	}
}
