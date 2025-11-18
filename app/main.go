package main

import (
	"fmt"
	"os"
	"bufio"
	"strings" 
	"slices"
)

// Ensures gofmt doesn't remove the "fmt" and "os" imports in stage 1 (feel free to remove this!)
var _ = fmt.Fprint
var _ = os.Stdout


func checkPermission(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	mode := info.Mode()
	return mode&0100 != 0
}

var builtin = []string{"echo", "type", "exit"}

func main() {
	// TODO: Uncomment the code below to pass the first stage

	for {
		fmt.Fprint(os.Stdout, "$ ")
		input, err := bufio.NewReader(os.Stdin).ReadString('\n')

		if err != nil {
			return
		}

		argv := strings.Fields(input)
		if len(argv) == 0 {
			continue
		}

		cmd := argv[0]

		switch cmd {
		case "exit":
			return
		case "echo":
			EchoCommand(argv)
		case "type":
			TypeCommand(argv)
		default:
			fmt.Fprintf(os.Stdout, "%s: command not found\n", cmd)
		}
		
	}
}

func EchoCommand(argv []string) {
	fmt.Fprintf(os.Stdout, "%s\n", strings.Join(argv[1:], " "))
}

func TypeCommand(argv []string) {

	if len(argv) == 1 {
		return
	}

	value := argv[1]

	if slices.Contains(builtin, value) {
		fmt.Fprintf(os.Stdout, "%s is a shell builtin\n", value)
		return
	}

	if file_path, exists := findFile(value); exists {
		fmt.Fprintf(os.Stdout, "%s is %s\n", value, file_path)
		return
	}

	fmt.Fprintf(os.Stdout, "%s: not found\n", value)

}

func findFile(value string) (string, bool) {
	PATH := os.Getenv("PATH")
	PATH_DIRS := strings.Split(PATH, ":")

	for _, dir := range PATH_DIRS {
		fullPath := dir + "/" + value
		if _, err := os.Stat(fullPath); err == nil {
			return fullPath, true
		}
	}

	return "", false
}

