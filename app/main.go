package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"sync"
	"io"
	"strconv"

	"github.com/chzyer/readline"
)

// Ensures gofmt doesn't remove the "fmt" and "os" imports in stage 1 (feel free to remove this!)
var _ = fmt.Fprint
var stdOut = os.Stdout
var stdIn = os.Stdin
var history = make([]string, 0)
func splitByPipe(command string) []string {
	commands := make([]string, 0)
	command = strings.TrimSpace(command)
	inSingleQuotes, inDoubleQuotes, isEscaped := false, false, false
	current := ""
	for _, char := range command {
		if inSingleQuotes {
			if char == '\'' {
				inSingleQuotes = false
			}
			current += string(char)
		} else if inDoubleQuotes {
			if char == '"' && !isEscaped {
				inDoubleQuotes = false
			} else if char == '\\' && !isEscaped {
				isEscaped = true
			} else {
				isEscaped = false
			}
			current += string(char)
		} else if isEscaped {
			current += string(char)
			isEscaped = false
		} else {
			switch char {
			case '\\':
				isEscaped = true
				current += string(char)
			case '\'':
				inSingleQuotes = true
				current += string(char)
			case '"':
				inDoubleQuotes = true
				current += string(char)
			case '|':
				commands = append(commands, current)
				current = ""
			default:
				current += string(char)
			}
		}

	}
	if current != "" {
		commands = append(commands, current)
	}
	return commands
}
func checkPermission(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	mode := info.Mode()
	return mode&0100 != 0
}

var builtin = []string{"echo", "type", "exit", "pwd", "cd", "history"}
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

func isAppendOperator(oprator string) bool {
	return oprator == ">>" || oprator == "1>>" || oprator == "2>>"
}

type AutoComplete struct {
	completer readline.AutoCompleter
	tabPress  bool
	rl        *readline.Instance
}

func getLongestPrefix(strs []string) string {
	longestPrefix := ""
	first, last := strs[0], strs[len(strs)-1]
	minLength := min(len(first), len(last))
	for i := 0; i < minLength; i++ {
		if first[i] == last[i] {
			longestPrefix += string(first[i])
		} else {
			break
		}
	}
	return longestPrefix
}
func (a *AutoComplete) Do(line []rune, pos int) (newLine [][]rune, length int) {
	newLine, length = a.completer.Do(line, pos)
	// if string(line) == "ech"{
	// 	fmt.Fprintf(os.Stdout, "new line is %v\n", newLine)
	// }

	sort.Slice(newLine, func(i, j int) bool {
		return string(newLine[i]) < string(newLine[j])
	})

	if len(newLine) == 0 {
		fmt.Fprintf(os.Stdout, "\x07")
		return nil, 0
	} else if len(newLine) == 1 {
		return newLine, length
	} else {
		strs := make([]string, 0, len(newLine))
		for _, s := range newLine {
			strs = append(strs, string(line)+strings.TrimSpace(string(s)))
		}
		longestPrefix := getLongestPrefix(strs)
		longestPrefix = longestPrefix[len(line):]
		if longestPrefix != "" {
			return [][]rune{[]rune(longestPrefix)}, len(longestPrefix)
		}

		if !a.tabPress {
			a.tabPress = true
			fmt.Fprintf(os.Stdout, "\x07")
			return nil, 0
		} else {
			a.tabPress = false
			fmt.Fprintf(os.Stdout, "\n%s\n", strings.Join(strs, "  "))
			a.rl.Refresh()
			return nil, 0
		}
	}
	// return newLine, length
}

func (a *AutoComplete) OnChange(line []rune, pos int, key rune) (newLine []rune, newPos int, ok bool) {
	if key != '\t' {
		a.tabPress = false
	}
	return nil, 0, false
}

func executeCommands(command string) {
	argv, outputFile, operator := parseCommand(command)
	if len(argv) == 0 {
		return
	}

	var f *os.File
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	if outputFile != "" {
		var err error
		var flags int

		if isAppendOperator(operator) {
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
		os.Exit(0)
	case "echo":
		EchoCommand(argv)
	case "type":
		TypeCommand(argv)
	case "pwd":
		pwdCommand(argv)
	case "cd":
		cdCommand(argv)
	case "history":
		historyCommand(argv)

	default:
		customCommand(argv)
	}

	if f != nil {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
		f.Close()
	}

}

func historyCommand(argv []string) {
	count := len(history)
	if len(argv) > 1 {
		count,_ = strconv.Atoi(argv[1])

	} 
	for i := len(history) - count;i < len(history); i++ {
		fmt.Fprintf(os.Stdout, "%d %s\n", i+1, history[i])
	}

}
func runCommand(argv []string, input io.Reader, output io.Writer, wg *sync.WaitGroup) {
	cmd := argv[0]
	wg.Add(1)
	go func(){
		defer wg.Done()
		if slices.Contains(builtin, cmd) {
			oldStdout := os.Stdout
			if w, ok := output.(*os.File); ok {
				os.Stdout = w
			}
			executeCommands(strings.Join(argv, " "))
			os.Stdout = oldStdout
		}else{
			cmd := exec.Command(cmd, argv[1:]...)
			cmd.Stdin = input
			cmd.Stdout = output
			cmd.Stderr = os.Stderr
			_ = cmd.Run()
		}
		if w, ok := output.(*os.File); ok && w != os.Stdout {
			w.Close()
		}
	}()
}

func main() {
	var items []readline.PrefixCompleterInterface
	PATH := os.Getenv("PATH")
	PATH_DIRS := strings.Split(PATH, ":")
	files := make([]string, 0)
	for _, dir := range PATH_DIRS {
		entry, _ := os.ReadDir(dir)

		for _, file := range entry {
			if file.IsDir() {
				continue
			}
			files = append(files, file.Name())
		}

	}

	seen := make(map[string]bool)
	for _, file := range files {
		if seen[file] {
			continue
		}
		seen[file] = true
		items = append(items, readline.PcItem(file))
	}

	for _, item := range builtin {
		if seen[item] {
			continue
		}
		seen[item] = true
		items = append(items, readline.PcItem(item))
	}

	var simpleCompleter = readline.NewPrefixCompleter(
		items...,
	)
	completer := &AutoComplete{
		completer: simpleCompleter,
		tabPress:  false,
	}

	rl, err := readline.NewEx(&readline.Config{
		Prompt:       "$ ",
		AutoComplete: completer,
		Listener:     completer,
	})
	completer.rl = rl
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	for {
		input, err := rl.Readline()
		if err != nil {
			break
		}
		history = append(history, input)
		commands := splitByPipe(input)
		if len(commands) > 1 {
			// reader, writer, err := os.Pipe()
			// if err != nil {
			// 	fmt.Fprintln(os.Stderr, "Error creating pipe:", err)
			// 	continue
			// }

			// command1, _, _ := parseCommand(commands[0])
			// command2, _, _ := parseCommand(commands[1])

			// var wg sync.WaitGroup
			
			// runCommand(command1, stdIn, writer, &wg)
			// runCommand(command2, reader, stdOut, &wg)

			// wg.Wait()
			// reader.Close()
			
			// continue
			prevInput := os.Stdin
			var wg sync.WaitGroup
			for i := 0;i < len(commands);i ++ {
				currentCommand,_,_ := parseCommand(commands[i])
				var reader *os.File
				var writer *os.File
				if i != len(commands) - 1{
					reader, writer,_ = os.Pipe()
				} else {
					writer = os.Stdout
				}
				runCommand(currentCommand, prevInput, writer, &wg)
				prevInput = reader

				

			}
			prevInput.Close()
			wg.Wait()
			continue
		}
		executeCommands(input)
		
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
