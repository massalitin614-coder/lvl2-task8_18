package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// хранит все параметры фильтрации
type options struct {
	after      int
	before     int
	count      bool
	ignoreCase bool
	invert     bool
	fixed      bool
	lineNum    bool
	pattern    string
	files      []string
}

func main() {
	opts, args, err := parseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "argument parsing error", err)
		os.Exit(1)
	}

	// Если после флагов остались аргументы, первый — шаблон, остальные — файлы
	if len(args) > 0 {
		opts.pattern = args[0]
		opts.files = args[1:]
	} else {
		// Пустой шаблон означает совпадение со всеми строками
		opts.pattern = ""
	}
	//если файлы не указаны, читаем из Stdin
	if len(opts.files) == 0 {
		if err := processFile(os.Stdin, opts, "stdin"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	//обрабатываем каждый файл
	for _, fname := range opts.files {
		file, err := os.Open(fname)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		err = processFile(file, opts, fname)
		file.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func parseFlags(args []string) (options, []string, error) {
	opts := options{}
	var leftovers []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			leftovers = append(leftovers, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			leftovers = append(leftovers, arg)
			continue
		}

		flagPart := arg[1:]

		// Обработка отдельного флага с аргументом (-A 5)
		if len(flagPart) == 1 && (flagPart[0] == 'A' || flagPart[0] == 'B' || flagPart[0] == 'C') {
			if i+1 >= len(args) {
				return opts, nil, fmt.Errorf("флаг -%c требует аргумент", flagPart[0])
			}
			valStr := args[i+1]
			val, err := strconv.Atoi(valStr)
			if err != nil || val < 0 {
				return opts, nil, fmt.Errorf("флаг -%c требует числовой аргумент", flagPart[0])
			}
			setContextFlag(&opts, flagPart[0], val)
			i++ // пропускаем значение
			continue
		}

		// Обработка слитного флага с числом (-A5)
		if len(flagPart) > 1 {
			first := flagPart[0]
			if first == 'A' || first == 'B' || first == 'C' {
				valStr := flagPart[1:]
				val, err := strconv.Atoi(valStr)
				if err != nil || val < 0 {
					return opts, nil, fmt.Errorf("флаг -%c требует числовой аргумент", first)
				}
				setContextFlag(&opts, first, val)
				continue
			}
		}

		// Обработка группы флагов без аргументов (-ivn)
		for _, ch := range flagPart {
			switch ch {
			case 'c':
				opts.count = true
			case 'i':
				opts.ignoreCase = true
			case 'v':
				opts.invert = true
			case 'F':
				opts.fixed = true
			case 'n':
				opts.lineNum = true
			case 'A', 'B', 'C':
				// Если дошли сюда, значит флаг требует аргумент, но он не был предоставлен
				return opts, nil, fmt.Errorf("флаг -%c требует аргумент", ch)
			default:
				return opts, nil, fmt.Errorf("неизвестный флаг -%c", ch)
			}
		}
	}
	return opts, leftovers, nil
}

// устанавливаем значение для -A, -B, -C
func setContextFlag(opts *options, flag byte, val int) {
	switch flag {
	case 'A':
		opts.after = val
	case 'B':
		opts.before = val
	case 'C':
		opts.after = val
		opts.before = val
	}
}

// возвращаем функцию проверки соответсвия строки шаблону
func compileMatcher(opts options) (func(string) bool, error) {
	pattern := opts.pattern
	if opts.fixed {
		//точное совпадение подстроки
		if opts.ignoreCase {
			pattern = strings.ToLower(pattern)
			return func(s string) bool {
				return strings.Contains(strings.ToLower(s), pattern)
			}, nil
		}

		return func(s string) bool {
			return strings.Contains(s, pattern)
		}, nil
	}

	//регулярное выражение
	if opts.ignoreCase {
		pattern = "(?i)" + pattern
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return re.MatchString, nil
}

// отрабатывает один входной поток
func processFile(r io.Reader, opts options, fname string) error {
	//читаем все строки
	var lines []string
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения %s: %v", fname, err)
	}

	//получаем функцию проверки
	matcher, err := compileMatcher(opts)
	if err != nil {
		return fmt.Errorf("ошибка компиляции шаблона")
	}

	//вычисляем совпадение
	matches := make([]bool, len(lines))
	for i, line := range lines {
		ok := matcher(line)
		if opts.invert {
			ok = !ok
		}
		matches[i] = ok
	}
	//режим подсчета
	if opts.count {
		count := 0
		for _, m := range matches {
			if m {
				count++
			}
		}
		fmt.Println(count)
		return nil
	}

	printWithContext(lines, matches, opts)
	return nil
}

func printWithContext(lines []string, matches []bool, opts options) {
	n := len(lines)
	output := make([]bool, n)
	for i, match := range matches {
		if match {
			start := i - opts.before
			if start < 0 {
				start = 0
			}
			end := i + opts.after
			if end >= n {
				end = n - 1
			}
			for j := start; j <= end; j++ {
				output[j] = true
			}
		}
	}

	firstGroup := true
	inGroup := false
	for i := 0; i < n; i++ {
		if output[i] {
			if !inGroup {
				// Начало новой группы
				if !firstGroup && (opts.after > 0 || opts.before > 0) {
					fmt.Println("--")
				}
				firstGroup = false
				inGroup = true
			}
			if opts.lineNum {
				fmt.Printf("%d:%s\n", i+1, lines[i])
			} else {
				fmt.Println(lines[i])
			}
		} else {
			inGroup = false
		}
	}
}
