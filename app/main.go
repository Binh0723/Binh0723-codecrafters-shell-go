package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
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
var operators = []string{">", "1>", "2>", ">>", "1>>", "2>>"}

func parseCommand(command string) ([]string, string, string) {
	res := make([]string, 0)
	command = strings.TrimSpace(command)
	printPath := ""
	inSingleQuotes, inDoubleQuotes, isEscaped, escapePossible, argHasQuote, isPrintPath := false, false, false, false, false, false
	current := ""
	operator := ""
	inArg := false

	for _, char := range command {

		if inSingleQuotes {
			if char == '\'' {
				inSingleQuotes = false
			} else {
				current += string(char)
			}
		} else if inDoubleQuotes {
			if char == '"' && !escapePossible {
				inDoubleQuotes = false
			} else if char == '\\' && !escapePossible {
				escapePossible = true
			} else {
				if escapePossible {
					if char == '"' || char == '\\' {
						current += string(char)
					} else {
						current += "\\" + string(char)
					}
					escapePossible = false
				} else {
					current += string(char)
				}
			}
		} else if isEscaped {
			current += string(char)
			isEscaped = false
		} else {
			switch char {
			case '\\':
				isEscaped = true
				argHasQuote = true
			case '\'':
				inSingleQuotes = true
				inArg = true
				argHasQuote = true
			case '"':
				inDoubleQuotes = true
				inArg = true
				argHasQuote = true
			case ' ':
				if inArg {
					if (slices.Contains(operators, current)) && !argHasQuote {
						operator = current
						isPrintPath = true
					} else {
						if isPrintPath {
							printPath = current
							isPrintPath = false
						} else {
							res = append(res, current)
						}
					}
					argHasQuote = false
					current = ""
					inArg = false
				}
			default:
				current += string(char)
				inArg = true
			}
		}
	}

	if inArg {
		if isPrintPath {
			printPath = current
		} else {
			res = append(res, current)
		}
	}
	return res, printPath, operator
}

func main() {
	// TODO: Uncomment the code below to pass the first stage

	for {
		fmt.Fprint(os.Stdout, "$ ")
		input, err := bufio.NewReader(os.Stdin).ReadString('\n')

		if err != nil {
			return
		}

		argv, outputFile, operator := parseCommand(input)
		if len(argv) == 0 {
			continue
		}

		var f *os.File
		oldStdout := os.Stdout
		oldStderr := os.Stderr

		if outputFile != "" {
			var err error
			var flags int

			if operator == ">>" || operator == "1>>" || operator == "2>>" {
				flags = os.O_APPEND | os.O_CREATE | os.O_WRONLY
			} else {
				flags = os.O_CREATE | os.O_WRONLY | os.O_TRUNC
			}

			f, err = os.OpenFile(outputFile, flags, 0644)
			if err != nil {
				fmt.Fprintln(os.Stdout, "Error opening file:", err)
			}

			switch operator {
			case ">", "1>", "1>>", ">>":
				os.Stdout = f
			case "2>", "2>>":
				os.Stderr = f
			}
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
		case "cat":
			catCommand(argv)
		default:
			customCommand(argv)
		}

		if f != nil {
			os.Stdout = oldStdout
			os.Stderr = oldStderr
			f.Close()
		}

	}
}

func catCommand(argv []string) {
	for _, arg := range argv[1:] {
		cmd := exec.Command("cat", arg)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
	}
}
func cdCommand(argv []string) {
	if len(argv) > 2 {
		fmt.Fprintf(os.Stderr, "%s: too many arguments\n", argv[0])
		return
	}

	path := argv[1]
	_, err := os.Stat(path)

	if err != nil && path != "~" {
		fmt.Fprintf(os.Stderr, "%s: %s: No such file or directory\n", argv[0], path)
		return
	}
	if argv[1] == "~" {
		home_path := os.Getenv("HOME")
		path = home_path
	}
	err = os.Chdir(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "can not change directory: %s\n", err)
	}

}

func pwdCommand(argv []string) {

	if len(argv) > 1 {
		fmt.Fprintf(os.Stdout, "%s: too many arguments\n", argv[0])
		return
	}
	dir, err := os.Getwd()

	if err != nil {
		fmt.Fprintf(os.Stdout, "%s: %s\n", argv[0], err)
		return
	}

	fmt.Fprintf(os.Stdout, "%s\n", dir)
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
				cmd.Stderr = os.Stderr

				err := cmd.Run()
				if err != nil {
					return
				}
				return
			}
		}
	}
	fmt.Fprintf(os.Stderr, "%s: command not found\n", value)
}
