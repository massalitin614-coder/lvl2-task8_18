package main

import (
	"fmt"
	"slices"
	"sort"
	"strings"
)

// anagrams находит группы анаграмм в заданном списке слов.
// Возвращает map, где ключом является первое встретившееся слово из группы (в нижнем регистре),
// а значением — отсортированный по алфавиту список всех слов группы (тоже в нижнем регистре).
// Группы, содержащие менее двух различных слов, исключаются.
// Слова в группе приводятся к нижнему регистру, дубликаты игнорируются.
func anagrams(words []string) map[string][]string {

	groups := make(map[string][]string)

	for _, word := range words {
		low := strings.ToLower(word)
		sig := runes(low)
		groups[sig] = append(groups[sig], low)
	}

	results := make(map[string][]string)

	for _, list := range groups {
		if len(list) < 2 {
			continue
		}

		first := list[0]
		sort.Strings(list)
		if list[0] == list[len(list)-1] {
			continue
		}
		results[first] = list
	}

	return results

}

// преобразуем строку в слайс рун, и возращаем обратно отсортированную строку
func runes(s string) string {

	runes := []rune(s)
	slices.Sort(runes)
	return string(runes)

}
func main() {
	sl := []string{"пятак", "тяпка", "листок", "слиток", "столик", "стол"}

	res := anagrams(sl)

	fmt.Println(res)
}
