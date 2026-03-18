package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"unicode"
)

// Unpack выполняет распаковку строки согласно условию.
// Возвращает распакованную строку или ошибку, если входные данные некорректны.
func Unpack(s string) (string, error) {
	runes := []rune(s)
	n := len(runes)
	var str strings.Builder
	i := 0
	for i < n {
		var curr rune
		if runes[i] == '\\' {

			if i+1 >= n {
				return "", fmt.Errorf("invalid escape at end of string")
			}
			curr = runes[i+1]
			i += 2
		} else if unicode.IsDigit(runes[i]) {
			return "", fmt.Errorf("digit without preceding character")
		} else {
			curr = runes[i]
			i++
		}
		//заводим второй индекс, который двигается по цифрам
		j := i

		for j < n && unicode.IsDigit(runes[j]) {
			j++
		}
		//переменная для количества повторений
		var count int
		//если цифр не было символ повторяется один раз
		if j == i {
			//1 символ
			count = 1
		} else {
			numStr := string(runes[i:j])
			var err error
			count, err = strconv.Atoi(numStr)
			if err != nil {
				return "", fmt.Errorf("string conversion error: %w", err)
			}
		}
		//записываем руну в строку
		if count > 0 {
			for k := 0; k < count; k++ {
				str.WriteRune(curr)
			}
		}
		//сдвигаемся, пропуская цифры
		i = j
	}
	return str.String(), nil
}

func main() {
	str := "qwe\\45"
	res, err := Unpack(str)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(res)
	fmt.Println(len(str), len(res))

}
