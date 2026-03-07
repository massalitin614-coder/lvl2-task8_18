package main

import (
	"testing"
)

func TestUnpack(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		wantErr  bool
	}{
		// Базовые примеры из задания
		{
			name:     "basic example 1",
			input:    "a4bc2d5e",
			expected: "aaaabccddddde",
			wantErr:  false,
		},
		{
			name:     "no numbers",
			input:    "abcd",
			expected: "abcd",
			wantErr:  false,
		},
		{
			name:     "only digits",
			input:    "45",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
			wantErr:  false,
		},

		// Дополнительное задание: escape-последовательности
		{
			name:     "escaped digits without repeat",
			input:    `qwe\4\5`,
			expected: "qwe45",
			wantErr:  false,
		},
		{
			name:     "escaped digit with following number",
			input:    `qwe\45`,
			expected: "qwe44444",
			wantErr:  false,
		},
		{
			name:     "escaped backslash without number",
			input:    `a\\b`,
			expected: `a\b`,
			wantErr:  false,
		},
		{
			name:     "only escaped backslash",
			input:    `\\`,
			expected: `\`,
			wantErr:  false,
		},
		{
			name:     "multiple escaped digits",
			input:    `\1\2\3`,
			expected: "123",
			wantErr:  false,
		},

		// Дополнительные граничные случаи
		{
			name:     "multidigit number",
			input:    "a12b",
			expected: "aaaaaaaaaaaab",
			wantErr:  false,
		},
		{
			name:     "zero repeat",
			input:    "a0b",
			expected: "b",
			wantErr:  false,
		},
		{
			name:     "unicode characters",
			input:    "ф4д2",
			expected: "ффффдд",
			wantErr:  false,
		},
		{
			name:     "digit after digit",
			input:    "a12b3",
			expected: "aaaaaaaaaaaabbb",
			wantErr:  false,
		},
		{
			name:     "escape at end",
			input:    `abc\`,
			expected: "",
			wantErr:  true,
		},
		{
			name:     "digit without preceding char after escape",
			input:    `\45`, // экранированная 4, затем 5 как число для неё
			expected: "44444",
			wantErr:  false,
		},
		{
			name:     "only escape with digit",
			input:    `\4`,
			expected: "4",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Unpack(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if got != tt.expected {
				t.Errorf("got %q, want %q", got, tt.expected)
			}
		})
	}
}
