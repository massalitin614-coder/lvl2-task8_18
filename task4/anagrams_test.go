package main

import (
	"reflect"
	"testing"
)

func TestAnagrams(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected map[string][]string
	}{
		{
			name:  "базовый пример из условия",
			input: []string{"пятак", "пятка", "тяпка", "листок", "слиток", "столик", "стол"},
			expected: map[string][]string{
				"пятак":  {"пятак", "пятка", "тяпка"},
				"листок": {"листок", "слиток", "столик"},
			},
		},
		{
			name:  "слова без анаграмм игнорируются",
			input: []string{"кот", "ток", "кар", "рак"},
			expected: map[string][]string{
				"кот": {"кот", "ток"},
				"кар": {"кар", "рак"},
			},
		},
		{
			name:     "нет анаграмм",
			input:    []string{"стол", "стул", "кресло"},
			expected: map[string][]string{},
		},
		{
			name:  "дубликаты сохраняются",
			input: []string{"пятак", "пятак", "тяпка"},
			expected: map[string][]string{
				"пятак": {"пятак", "пятак", "тяпка"},
			},
		},
		{
			name:  "разный регистр",
			input: []string{"Пятак", "пятка", "Тяпка"},
			expected: map[string][]string{
				"пятак": {"пятак", "пятка", "тяпка"},
			},
		},
		{
			name:     "группа из двух одинаковых слов исключается",
			input:    []string{"пятак", "пятак"},
			expected: map[string][]string{},
		},
		{
			name:     "пустой вход",
			input:    []string{},
			expected: map[string][]string{},
		},
		{
			name:     "одно слово без анаграмм",
			input:    []string{"один"},
			expected: map[string][]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := anagrams(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("anagrams() = %v, expected %v", got, tt.expected)
			}
		})
	}
}

func TestRunes(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"пятак", "акптя"},
		{"тяпка", "акптя"},
		{"листок", "иклост"},
		{"столик", "иклост"},
		{"", ""},
		{"a", "a"},
		{"ba", "ab"},
		{"cba", "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := runes(tt.input); got != tt.expected {
				t.Errorf("runes(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func BenchmarkAnagrams(b *testing.B) {
	words := []string{"пятак", "пятка", "тяпка", "листок", "слиток", "столик", "стол"}
	for i := 0; i < b.N; i++ {
		anagrams(words)
	}
}
