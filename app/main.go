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

	builtins := map[string]bool {
		"echo": true,
		"type": true,
		"exit": true,
	}
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
		} else if parts[0] == "type"{
			if builtins[parts[1]] {
				fmt.Fprintf(os.Stdout, "%s is a shell builtin\n", parts[1])
			} else {
				fmt.Fprintf(os.Stdout, "%s: not found\n", parts[1])
			}
		}else {
			fmt.Fprintf(os.Stdout, "%s: command not found\n", command)
		}	
	}
}
