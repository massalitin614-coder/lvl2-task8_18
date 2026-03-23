package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

var currentCmd *exec.Cmd

type Command struct {
	Args   []string
	Stdin  string
	Stdout string
	Append bool
}

// tokenize — без изменений (см. предыдущий код)
func tokenize(s string) []string {
	var tokens []string
	var cur strings.Builder
	inQuote := false
	quoteChar := byte(0)

	for i := 0; i < len(s); i++ {
		ch := s[i]
		if (ch == '"' || ch == '\'') && !inQuote {
			inQuote = true
			quoteChar = ch
			cur.WriteByte(ch)
			continue
		}
		if inQuote && ch == quoteChar {
			inQuote = false
			quoteChar = 0
			cur.WriteByte(ch)
			continue
		}
		if inQuote {
			cur.WriteByte(ch)
			continue
		}

		if ch == '|' || ch == '>' || ch == '<' || ch == '&' {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			if ch == '&' && i+1 < len(s) && s[i+1] == '&' {
				tokens = append(tokens, "&&")
				i++
			} else if ch == '|' && i+1 < len(s) && s[i+1] == '|' {
				tokens = append(tokens, "||")
				i++
			} else {
				tokens = append(tokens, string(ch))
			}
			continue
		}

		if ch == ' ' || ch == '\t' {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
			continue
		}

		cur.WriteByte(ch)
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func expandEnv(tokens []string) []string {
	result := make([]string, 0, len(tokens))
	for _, tok := range tokens {
		if (strings.HasPrefix(tok, `"`) && strings.HasSuffix(tok, `"`)) ||
			(strings.HasPrefix(tok, `'`) && strings.HasSuffix(tok, `'`)) {
			tok = tok[1 : len(tok)-1]
		}
		tok = os.ExpandEnv(tok)
		result = append(result, tok)
	}
	return result
}

// parsePipeline разбивает токены на пайплайны по операторам && и ||
func parsePipeline(tokens []string) ([][]Command, string) {
	var pipelines [][]Command
	var op string
	var group []string

	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		if tok == "&&" || tok == "||" {
			if op != "" && op != tok {
				// смешанные операторы не поддерживаем, игнорируем
			}
			if len(group) > 0 {
				pipeline := parsePipelineGroup(group)
				if pipeline != nil {
					pipelines = append(pipelines, pipeline)
				}
				group = nil
			}
			op = tok
		} else {
			group = append(group, tok)
		}
	}
	if len(group) > 0 {
		pipeline := parsePipelineGroup(group)
		if pipeline != nil {
			pipelines = append(pipelines, pipeline)
		}
	}
	return pipelines, op
}

// parsePipelineGroup парсит группу токенов (без && и ||) в пайплайн
func parsePipelineGroup(tokens []string) []Command {
	var pipeline []Command
	var curCmd Command
	var curArgs []string

	i := 0
	for i < len(tokens) {
		tok := tokens[i]
		switch tok {
		case "|":
			if len(curArgs) == 0 {
				return nil
			}
			curCmd.Args = curArgs
			pipeline = append(pipeline, curCmd)
			curArgs = nil
			curCmd = Command{}
			i++

		case ">", ">>":
			if i+1 >= len(tokens) {
				return nil
			}
			if curCmd.Stdout != "" {
				return nil
			}
			curCmd.Stdout = tokens[i+1]
			curCmd.Append = (tok == ">>")
			i += 2

		case "<":
			if i+1 >= len(tokens) {
				return nil
			}
			if curCmd.Stdin != "" {
				return nil
			}
			curCmd.Stdin = tokens[i+1]
			i += 2

		default:
			curArgs = append(curArgs, tok)
			i++
		}
	}
	if len(curArgs) > 0 {
		curCmd.Args = curArgs
		pipeline = append(pipeline, curCmd)
	}
	return pipeline
}

func isBuiltin(cmd Command) bool {
	if len(cmd.Args) == 0 {
		return false
	}
	name := cmd.Args[0]
	return name == "cd" || name == "pwd" || name == "echo" || name == "kill" || name == "ps"
}

func execBuiltin(cmd Command) (int, error) {
	switch cmd.Args[0] {
	case "cd":
		var dir string
		if len(cmd.Args) == 1 {
			dir, _ = os.UserHomeDir()
		} else {
			dir = cmd.Args[1]
		}
		err := os.Chdir(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cd: %v\n", err)
			return 1, nil
		}
		return 0, nil
	case "pwd":
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "pwd: %v\n", err)
			return 1, nil
		}
		fmt.Println(dir)
		return 0, nil
	case "echo":
		fmt.Println(strings.Join(cmd.Args[1:], " "))
		return 0, nil
	case "kill":
		if len(cmd.Args) < 2 {
			fmt.Fprintln(os.Stderr, "kill: missing PID")
			return 1, nil
		}
		pid, err := strconv.Atoi(cmd.Args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "kill: invalid PID: %v\n", err)
			return 1, nil
		}
		proc, err := os.FindProcess(pid)
		if err != nil {
			fmt.Fprintf(os.Stderr, "kill: %v\n", err)
			return 1, nil
		}
		err = proc.Signal(syscall.SIGTERM)
		if err != nil {
			fmt.Fprintf(os.Stderr, "kill: %v\n", err)
			return 1, nil
		}
		return 0, nil
	case "ps":
		cmd := exec.Command("ps", "aux")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			return 1, nil
		}
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown builtin")
	}
}

func runCommand(cmd Command) (int, error) {
	if len(cmd.Args) == 0 {
		return 0, nil
	}
	c := exec.Command(cmd.Args[0], cmd.Args[1:]...)

	if cmd.Stdin != "" {
		f, err := os.Open(cmd.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot open %s: %v\n", cmd.Stdin, err)
			return 1, err
		}
		defer f.Close()
		c.Stdin = f
	} else {
		c.Stdin = os.Stdin
	}

	if cmd.Stdout != "" {
		var f *os.File
		var err error
		if cmd.Append {
			f, err = os.OpenFile(cmd.Stdout, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		} else {
			f, err = os.Create(cmd.Stdout)
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "cannot create %s: %v\n", cmd.Stdout, err)
			return 1, err
		}
		defer f.Close()
		c.Stdout = f
	} else {
		c.Stdout = os.Stdout
	}
	c.Stderr = os.Stderr

	currentCmd = c
	err := c.Run()
	currentCmd = nil

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode(), nil
		}
		return 1, err
	}
	return 0, nil
}

func runPipeline(cmds []Command) (int, error) {
	if len(cmds) == 0 {
		return 0, nil
	}
	if len(cmds) == 1 {
		return runCommand(cmds[0])
	}

	var stdin io.Reader = os.Stdin
	var lastCmd *exec.Cmd
	var lastErr error

	for i, cmd := range cmds {
		pr, pw := io.Pipe()
		c := exec.Command(cmd.Args[0], cmd.Args[1:]...)
		c.Stdin = stdin
		if i == len(cmds)-1 {
			c.Stdout = os.Stdout
		} else {
			c.Stdout = pw
		}
		c.Stderr = os.Stderr

		if err := c.Start(); err != nil {
			lastErr = err
			break
		}
		go func(pw *io.PipeWriter) {
			c.Wait()
			pw.Close()
		}(pw)

		stdin = pr
		lastCmd = c
	}

	if lastCmd != nil && lastErr == nil {
		err := lastCmd.Wait()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				lastErr = fmt.Errorf("exit code %d", exitErr.ExitCode())
			} else {
				lastErr = err
			}
		}
	}

	if lastErr != nil {
		return 1, lastErr
	}
	return 0, nil
}

func executeLine(line string) {
	tokens := tokenize(line)
	if len(tokens) == 0 {
		return
	}
	tokens = expandEnv(tokens)

	pipelines, op := parsePipeline(tokens)
	if pipelines == nil {
		fmt.Fprintln(os.Stderr, "syntax error")
		return
	}

	for i, pipeline := range pipelines {
		code, err := runPipeline(pipeline)
		if err != nil && code == 0 {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		if op == "&&" && code != 0 {
			break
		}
		if op == "||" && code == 0 {
			break
		}
		if i == len(pipelines)-1 {
			break
		}
	}
}

func signalHandler() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			if currentCmd != nil && currentCmd.Process != nil {
				currentCmd.Process.Signal(os.Interrupt)
			}
		}
	}()
}

func main() {
	signalHandler()
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("$ ")
		if !scanner.Scan() {
			break
		}
		line := scanner.Text()
		if line == "" {
			continue
		}
		executeLine(line)
	}
	fmt.Println()
}
