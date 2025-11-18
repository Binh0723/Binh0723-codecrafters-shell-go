package main

import (
	"fmt"
	"os"
	"bufio"
	"strings" 
	"slices"
	"path/filepath"
	"os/exec"
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

var builtin = []string{"echo", "type", "exit", "pwd", "cd"}

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
		case "pwd":
			pwdCommand(argv)
		case "cd":
			cdCommand(argv)
		default:
			customCommand(argv)
		}
		
	}
}


func cdCommand(argv []string) {
	if len(argv) > 2 {
		fmt.Fprintf(os.Stdout, "%s: too many arguments\n", argv[0])
		return
	}
	
	path := argv[1]
	_, err := os.Stat(path)

	if err != nil {
		fmt.Fprintf(os.Stdout, "%s: %s: No such file or directory\n", argv[0],path)
		return
	} else {
		_ := os.Chdir(path)
	}
}

func pwdCommand(argv []string) {

	if len(argv) > 1{
		fmt.Fprintf(os.Stdout, "%s: too many arguments\n", argv[0])
		return
	}
	dir, err := os.Getwd()

	if err != nil {
		fmt.Fprintf(os.Stdout, "%s: %s\n", argv[0], err)
		return
	}

	fmt.Fprintf(os.Stdout, "%s\n",dir)
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
		
		fullPath := filepath.Join(dir, value)
		if info, err := os.Stat(fullPath); err == nil {
			if !info.IsDir() && checkPermission(fullPath) {
				return fullPath, true
			}
		}
	}

	return "", false
}

func customCommand(argv []string) {
	value := argv[0]

	PATH := os.Getenv("PATH")
	PATH_DIRS := strings.Split(PATH, ":")

	for _, dir := range PATH_DIRS {
		fullPath := filepath.Join(dir, value)
		if info, err := os.Stat(fullPath); err == nil {
			if !info.IsDir() && checkPermission(fullPath) {
				cmd := exec.Command(value, argv[1:]...)
				cmd.Stdout = os.Stdout

				err := cmd.Run()
				if err != nil {
					return
				}
				return
			}
		}
	}
	fmt.Fprintf(os.Stdout, "%s: command not found\n", value)
}