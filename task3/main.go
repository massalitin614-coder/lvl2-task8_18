package main

// Импортируем необходимые стандартные пакеты
import (
	"bufio"   // для построчного чтения
	"fmt"     // для форматированного вывода
	"os"      // для работы с файлами и стандартным вводом/выводом
	"sort"    // для сортировки срезов
	"strconv" // для преобразования строк в числа
	"strings" // для работы со строками (разбиение, обрезка)
)

// Line представляет одну строку входных данных и её ключ сортировки.
type Line struct {
	Text string // исходный текст строки
	Key  any    // вычисленный ключ (float64, int или string) для сравнения
}

// Flags хранит все опции, переданные через командную строку.
type Flags struct {
	keyColumn int      // номер колонки для сортировки (1-...), 0 = вся строка
	numeric   bool     // -n: числовая сортировка
	reverse   bool     // -r: обратный порядок
	unique    bool     // -u: только уникальные строки (по ключу)
	month     bool     // -M: сортировка по названию месяца
	trimSpace bool     // -b: игнорировать пробелы в начале и конце строки/колонки
	check     bool     // -c: проверить, отсортированы ли данные
	human     bool     // -h: сортировка с человекочитаемыми суффиксами (K, M, G...)
	files     []string // список файлов для чтения (пустой = stdin)
}

// monthMap сопоставляет трёхбуквенные названия месяцев (в нижнем регистре) с их номерами.
var monthMap = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
	"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

// humanSuffixes определяет множители для человекочитаемых суффиксов.
// Ключ — байт суффикса (в верхнем регистре), значение — множитель (степень 1024).
var humanSuffixes = map[byte]float64{
	'K': 1024,
	'M': 1024 * 1024,
	'G': 1024 * 1024 * 1024,
	'T': 1024 * 1024 * 1024 * 1024,
	'P': 1024 * 1024 * 1024 * 1024 * 1024,
	'E': 1024 * 1024 * 1024 * 1024 * 1024 * 1024,
}

// parseArgs разбирает аргументы командной строки и возвращает заполненную структуру Flags.
func parseArgs() (*Flags, error) {
	// Создаём структуру с начальными значениями (keyColumn = 0 по умолчанию)
	flags := &Flags{keyColumn: 0}
	// Получаем аргументы, начиная с первого (после имени программы)
	args := os.Args[1:]

	// Проходим по всем аргументам
	for i := 0; i < len(args); i++ {
		arg := args[i] // текущий аргумент

		// Если встречаем "--", то все последующие аргументы считаем именами файлов
		if arg == "--" {
			// Добавляем оставшиеся аргументы в список файлов и выходим из цикла
			flags.files = append(flags.files, args[i+1:]...)
			break
		}

		// Если аргумент не начинается с '-', значит это имя файла
		if !strings.HasPrefix(arg, "-") {
			flags.files = append(flags.files, arg)
			continue
		}

		// Обработка флага -k с отдельным аргументом (например, "-k 2")
		if arg == "-k" {
			// Проверяем, что после -k есть ещё аргумент
			if i+1 >= len(args) {
				return nil, fmt.Errorf("option -k requires an argument")
			}
			i++ // переходим к следующему аргументу, который содержит номер колонки
			// Преобразуем строку в число
			num, err := strconv.Atoi(args[i])
			if err != nil || num < 1 {
				return nil, fmt.Errorf("invalid column number: %s", args[i])
			}
			flags.keyColumn = num // сохраняем номер колонки
			continue
		}

		// Обработка флага -k, записанного слитно (например, "-k2")
		if strings.HasPrefix(arg, "-k") && len(arg) > 2 {
			numStr := arg[2:]                // часть после "-k"
			num, err := strconv.Atoi(numStr) // преобразуем в число
			if err != nil || num < 1 {
				return nil, fmt.Errorf("invalid column number: %s", numStr)
			}
			flags.keyColumn = num // сохраняем номер колонки
			continue
		}

		// Разбор комбинированных флагов (например, "-nr")
		// Проходим по каждому символу после первого '-'
		for _, ch := range arg[1:] {
			switch ch {
			case 'n':
				flags.numeric = true
			case 'r':
				flags.reverse = true
			case 'u':
				flags.unique = true
			case 'M':
				flags.month = true
			case 'b':
				flags.trimSpace = true
			case 'c':
				flags.check = true
			case 'h':
				flags.human = true
			default:

				// Если встретили неизвестный флаг, возвращаем ошибку
				return nil, fmt.Errorf("unknown option: -%c", ch)
			}
		}
	}
	// Возвращаем заполненную структуру и nil в качестве ошибки
	return flags, nil
}

// readLines читает все строки из указанных файлов (или stdin, если файлы не заданы)
// и возвращает срез Line с незаполненными ключами.
func readFiles(files []string) ([]Line, error) {
	var lines []Line // срез для хранения строк

	// Если файлы не указаны, читаем из стандартного ввода (обозначается "-")
	if len(files) == 0 {
		files = []string{"-"} // stdin
	}

	// Перебираем все имена файлов
	for _, name := range files {
		var f *os.File
		var err error

		// Если имя "-", используем os.Stdin, иначе открываем файл
		if name == "-" {
			f = os.Stdin
		} else {
			f, err = os.Open(name)
			if err != nil {
				return nil, err // не удалось открыть файл
			}
			// Откладываем закрытие файла до выхода из функции (после чтения)
			defer f.Close()
		}

		// Создаём сканер для построчного чтения
		scanner := bufio.NewScanner(f)
		const maxCapacity = 1024 * 1024
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, maxCapacity)
		// Читаем все строки из файла
		for scanner.Scan() {
			// Добавляем строку в срез (ключ пока не вычислен)
			lines = append(lines, Line{Text: scanner.Text()})
		}
		// Проверяем, не возникла ли ошибка при сканировании
		if err := scanner.Err(); err != nil {
			return nil, err
		}
	}
	// Возвращаем все прочитанные строки
	return lines, nil
}

// parseNumeric преобразует строку в float64; при ошибке возвращает 0.
func parseNumeric(s string) float64 {
	// Пытаемся преобразовать строку в число с плавающей точкой
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		// Если не удалось, возвращаем 0 (как в GNU sort)
		return 0.0
	}
	return v
}

// parseMonth преобразует строку в номер месяца (1-12); при ошибке возвращает 0.
func parseMonth(s string) int {
	// Убираем возможные пробелы по краям
	s = strings.TrimSpace(s)
	// Если длина меньше 3, не можем определить месяц
	if len(s) < 3 {
		return 0
	}
	// Берём первые три символа и приводим к нижнему регистру
	key := strings.ToLower(s[:3])
	// Ищем в map monthMap; если найдено, возвращаем номер месяца
	if num, ok := monthMap[key]; ok {
		return num
	}
	// Иначе возвращаем 0 (неизвестный месяц)
	return 0
}

// parseHuman преобразует строку вида "1.5K" в число байт (float64); при ошибке возвращает 0.
func parseHuman(s string) float64 {
	// Убираем пробелы
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	// Определяем последний символ строки
	last := s[len(s)-1]
	var suffix byte    // суффикс (буква)
	var numPart string // числовая часть
	// Если последний символ — буква (латинская), считаем его суффиксом
	if last >= 'A' && last <= 'Z' || last >= 'a' && last <= 'z' {
		suffix = last
		// Приводим суффикс к верхнему регистру для поиска в humanSuffixes
		if suffix >= 'a' && suffix <= 'z' {
			suffix -= 'a' - 'A'
		}
		// Числовая часть — всё, кроме последнего символа
		numPart = s[:len(s)-1]
	} else {
		// Суффикса нет, вся строка — число
		numPart = s
	}
	// Преобразуем числовую часть в float64
	v, err := strconv.ParseFloat(numPart, 64)
	if err != nil {
		return 0 // при ошибке парсинга возвращаем 0
	}
	// Если есть суффикс, умножаем на соответствующий множитель
	if mult, ok := humanSuffixes[suffix]; ok {
		v *= mult
	}
	return v
}

// computeKeys заполняет поле Key для каждой строки в соответствии с флагами.
func computeKeys(lines []Line, flags *Flags) {
	// Проходим по всем строкам (по индексу, чтобы изменять оригинальный срез)
	for i := range lines {
		var keyStr string // строка-кандидат для ключа (до преобразования)

		// Если указана сортировка по колонке (keyColumn > 0)
		if flags.keyColumn > 0 {
			// Разделяем строку по табуляции
			parts := strings.Split(lines[i].Text, "\t")
			// Проверяем, существует ли колонка с нужным номером (индексация с 0)
			if flags.keyColumn-1 < len(parts) {
				keyStr = parts[flags.keyColumn-1]
			} else {
				// Если колонки нет, ключ — пустая строка
				keyStr = ""
			}
		} else {
			// Сортировка по всей строке
			keyStr = lines[i].Text
		}
		// Если флаг -b установлен, обрезаем пробелы в начале и конце
		if flags.trimSpace {
			keyStr = strings.TrimSpace(keyStr)
		}

		// Преобразуем keyStr в нужный тип в зависимости от флагов
		switch {
		case flags.human:
			// Человекочитаемый размер -> float64
			lines[i].Key = parseHuman(keyStr)
		case flags.numeric:
			// Числовая сортировка -> float64
			lines[i].Key = parseNumeric(keyStr)
		case flags.month:
			// Сортировка по месяцу -> int (номер месяца)
			lines[i].Key = parseMonth(keyStr)
		default:
			// Обычная строковая сортировка -> string
			lines[i].Key = keyStr
		}
	}
}

// less возвращает true, если строка i должна идти раньше строки j.
func less(i, j Line, flags *Flags) bool {
	var result bool // результат сравнения "i меньше j" без учёта reverse

	// Определяем тип ключа и сравниваем соответствующие значения
	switch {
	case flags.human, flags.numeric:
		// Для float64
		a := i.Key.(float64)
		b := j.Key.(float64)
		result = a < b
	case flags.month:
		// Для int (номер месяца)
		a := i.Key.(int)
		b := j.Key.(int)
		result = a < b
	default:
		// Для string
		a := i.Key.(string)
		b := j.Key.(string)
		result = a < b
	}

	// Если флаг reverse установлен, инвертируем результат
	if flags.reverse {
		return !result
	}
	return result
}

// equalKeys проверяет равенство ключей двух строк.
func equalKeys(a, b Line) bool {
	// Используем type switch для сравнения в зависимости от типа ключа
	switch a.Key.(type) {
	case float64:
		return a.Key.(float64) == b.Key.(float64)
	case int:
		return a.Key.(int) == b.Key.(int)
	case string:
		return a.Key.(string) == b.Key.(string)
	default:
		// На случай, если тип не совпадает (не должно происходить)
		return false
	}
}

// checkSorted проверяет, упорядочены ли строки согласно флагам.
// Возвращает true, если порядок соблюдён, иначе false и индекс первого нарушения.
func checkSorted(lines []Line, flags *Flags) (bool, int) {
	// Проходим по всем парам соседних строк
	for i := 0; i < len(lines)-1; i++ {
		// Проверяем, должна ли следующая строка идти раньше текущей (нарушение порядка)
		// less(lines[i+1], lines[i], flags) вернёт true, если lines[i+1] < lines[i]
		if less(lines[i+1], lines[i], flags) {
			// Нарушение порядка на индексе i+1
			return false, i + 1
		}
	}
	// Все строки упорядочены
	return true, -1
}

// unique удаляет дубликаты из отсортированного среза lines (по ключам) и возвращает новый срез.
func unique(lines []Line) []Line {
	// Если срез пуст, возвращаем его же
	if len(lines) == 0 {
		return lines
	}
	// Создаём новый срез и добавляем первую строку
	res := []Line{lines[0]}
	// Проходим по остальным строкам, начиная со второй
	for i := 1; i < len(lines); i++ {
		// Если ключ текущей строки не равен ключу предыдущей добавленной, добавляем её
		if !equalKeys(lines[i], lines[i-1]) {
			res = append(res, lines[i])
		}
	}
	return res
}

func main() {
	// 1. Разбор аргументов командной строки
	flags, err := parseArgs()
	if err != nil {
		// В случае ошибки выводим сообщение в stderr и завершаем с кодом 2
		fmt.Fprintf(os.Stderr, "sort: %v\n", err)
		os.Exit(2)
	}

	// 2. Чтение всех строк из указанных файлов (или stdin)
	lines, err := readFiles(flags.files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sort: %v\n", err)
		os.Exit(2)
	}

	// 3. Вычисление ключей сортировки для всех строк
	computeKeys(lines, flags)

	// 4. Если установлен флаг -c (проверка на упорядоченность)
	if flags.check {
		ok, idx := checkSorted(lines, flags)
		if !ok {
			// Если порядок нарушен, выводим диагностику и завершаем с кодом 1
			fmt.Fprintf(os.Stderr, "sort: input disorder: %s\n", lines[idx].Text)
			os.Exit(1)
		}
		// Если порядок соблюдён, просто завершаем без вывода
		return
	}

	// 5. Сортировка среза lines с использованием функции less
	sort.Slice(lines, func(i, j int) bool {
		return less(lines[i], lines[j], flags)
	})

	// 6. Если установлен флаг -u, удаляем дубликаты
	if flags.unique {
		lines = unique(lines)
	}

	// 7. Вывод отсортированных (и уникальных) строк
	for _, line := range lines {
		fmt.Println(line.Text)
	}
}
