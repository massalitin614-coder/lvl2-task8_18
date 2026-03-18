package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Config хранит настройки из флагов
type Config struct {
	Fields    string
	Delimiter string
	Separated bool
}

func main() {
	cfg := parseFlags()
	if cfg.Fields == "" {
		fmt.Fprintln(os.Stderr, "не указаны поля (-f)")
		os.Exit(1)
	}
	if err := run(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "ошибка: %v\n", err)
		os.Exit(1)
	}
}

// parseFlags считывает флаги командной строки
func parseFlags() Config {
	var cfg Config
	flag.StringVar(&cfg.Fields, "f", "", "выбираемые поля (например 1,3-5)")
	flag.StringVar(&cfg.Delimiter, "d", "\t", "разделитель полей")
	flag.BoolVar(&cfg.Separated, "s", false, "только строки с разделителем")
	flag.Parse()
	return cfg
}

// parseFields превращает строку "1,3-5" в map[int]bool {1:true, 3:true, 4:true, 5:true}
func parseFields(fieldStr string) (map[int]bool, error) {
	fields := make(map[int]bool)
	parts := strings.Split(fieldStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			// Диапазон, например "3-5"
			bounds := strings.Split(part, "-")
			if len(bounds) != 2 {
				return nil, fmt.Errorf("некорректный диапазон: %s", part)
			}
			start, err1 := strconv.Atoi(bounds[0])
			end, err2 := strconv.Atoi(bounds[1])
			if err1 != nil || err2 != nil || start < 1 || end < 1 || start > end {
				return nil, fmt.Errorf("некорректный диапазон чисел: %s", part)
			}
			for i := start; i <= end; i++ {
				fields[i] = true
			}
		} else {
			// Одиночное поле
			num, err := strconv.Atoi(part)
			if err != nil || num < 1 {
				return nil, fmt.Errorf("некорректный номер поля: %s", part)
			}
			fields[num] = true
		}
	}
	return fields, nil
}

// run основная логика обработки
func run(cfg Config) error {
	// Получаем map нужных полей
	fields, err := parseFields(cfg.Fields)
	if err != nil {
		return err
	}

	// Превращаем map в отсортированный список номеров полей (один раз)
	nums := make([]int, 0, len(fields))
	for n := range fields {
		nums = append(nums, n)
	}
	sort.Ints(nums)

	scanner := bufio.NewScanner(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	for scanner.Scan() {
		line := scanner.Text()
		// Разделяем строку на части
		parts := strings.Split(line, cfg.Delimiter)

		// Если включен флаг -s и в строке нет разделителя (длина частей == 1), пропускаем
		if cfg.Separated && len(parts) == 1 {
			continue
		}

		// Собираем выбранные поля
		selected := make([]string, 0, len(nums))
		for _, n := range nums {
			idx := n - 1 // индекс в срезе (нумерация с 0)
			if idx < len(parts) {
				selected = append(selected, parts[idx])
			}
		}

		// Выводим, если что-то выбрали
		if len(selected) > 0 {
			fmt.Fprintln(writer, strings.Join(selected, cfg.Delimiter))
		}
	}

	return scanner.Err()
}
