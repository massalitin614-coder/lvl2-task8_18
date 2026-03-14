package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestParseFlags проверяет разбор флагов.
func TestParseFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    options
		left    []string
		wantErr string
	}{
		{
			name: "простой флаг",
			args: []string{"-i", "pattern"},
			want: options{ignoreCase: true},
			left: []string{"pattern"},
		},
		{
			name: "группа флагов",
			args: []string{"-ivn", "pat"},
			want: options{ignoreCase: true, invert: true, lineNum: true},
			left: []string{"pat"},
		},
		{
			name: "контекст раздельно",
			args: []string{"-A", "2", "-B", "1", "pattern"},
			want: options{after: 2, before: 1},
			left: []string{"pattern"},
		},
		{
			name: "контекст слитно",
			args: []string{"-A5", "-B2", "p"},
			want: options{after: 5, before: 2},
			left: []string{"p"},
		},
		{
			name:    "неизвестный флаг",
			args:    []string{"-x"},
			wantErr: "неизвестный флаг -x",
		},
		{
			name:    "флаг без значения",
			args:    []string{"-A"},
			wantErr: "флаг -A требует аргумент",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, left, err := parseFlags(tt.args)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("ожидалась ошибка %q, получено %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got.after != tt.want.after || got.before != tt.want.before ||
				got.count != tt.want.count || got.ignoreCase != tt.want.ignoreCase ||
				got.invert != tt.want.invert || got.fixed != tt.want.fixed ||
				got.lineNum != tt.want.lineNum {
				t.Errorf("options:\n got %+v\nwant %+v", got, tt.want)
			}
			if !slicesEqual(left, tt.left) {
				t.Errorf("leftover args: got %v, want %v", left, tt.left)
			}
		})
	}
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestCompileMatcher проверяет функцию проверки строк.
func TestCompileMatcher(t *testing.T) {
	tests := []struct {
		name    string
		opts    options
		line    string
		want    bool
		wantErr bool
	}{
		{
			name: "фиксированная строка",
			opts: options{pattern: "foo", fixed: true},
			line: "foobar",
			want: true,
		},
		{
			name: "фиксированная с игнором регистра",
			opts: options{pattern: "foo", fixed: true, ignoreCase: true},
			line: "FOO",
			want: true,
		},
		{
			name: "регулярное выражение",
			opts: options{pattern: "^f.*o$"},
			line: "faro",
			want: true,
		},
		{
			name: "регулярное с игнором регистра",
			opts: options{pattern: "^foo", ignoreCase: true},
			line: "FOObar",
			want: true,
		},
		{
			name: "пустой шаблон (все строки)",
			opts: options{pattern: ""},
			line: "anything",
			want: true,
		},
		{
			name:    "некорректное регулярное выражение",
			opts:    options{pattern: "["},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := compileMatcher(tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got := matcher(tt.line); got != tt.want {
				t.Errorf("matcher(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

// helper для захвата вывода функции, печатающей в stdout.
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	return buf.String()
}

// TestProcessFile проверяет обработку файла.
func TestProcessFile(t *testing.T) {
	content := []string{
		"line one",
		"line two with foo",
		"line three",
		"another foo here",
		"last line",
	}
	tmp := t.TempDir()
	fname := filepath.Join(tmp, "test.txt")
	if err := os.WriteFile(fname, []byte(strings.Join(content, "\n")), 0644); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name     string
		opts     options
		useStdin string // если не пусто, читать из этой строки вместо файла
		expected string
	}{
		{
			name:     "обычный поиск",
			opts:     options{pattern: "foo"},
			expected: "line two with foo\nanother foo here\n",
		},
		{
			name:     "с номерами строк",
			opts:     options{pattern: "foo", lineNum: true},
			expected: "2:line two with foo\n4:another foo here\n",
		},
		{
			name:     "инвертирование",
			opts:     options{pattern: "foo", invert: true},
			expected: "line one\nline three\nlast line\n",
		},
		{
			name:     "подсчёт",
			opts:     options{pattern: "foo", count: true},
			expected: "2\n",
		},
		{
			name:     "контекст после (слияние групп)",
			opts:     options{pattern: "foo", after: 1},
			expected: "line two with foo\nline three\nanother foo here\nlast line\n",
		},
		{
			name:     "контекст до (слияние групп)",
			opts:     options{pattern: "foo", before: 1},
			expected: "line one\nline two with foo\nline three\nanother foo here\n",
		},
		{
			name:     "контекст вокруг (слияние групп)",
			opts:     options{pattern: "foo", after: 1, before: 1},
			expected: "line one\nline two with foo\nline three\nanother foo here\nlast line\n",
		},
		{
			name:     "чтение из stdin",
			opts:     options{pattern: "hello"},
			useStdin: "hello world\nbye world\nhello again\n",
			expected: "hello world\nhello again\n",
		},
		{
			name:     "пустой шаблон",
			opts:     options{pattern: ""},
			expected: strings.Join(content, "\n") + "\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r *os.File
			if tt.useStdin != "" {
				tmpStdin, err := os.CreateTemp("", "stdin")
				if err != nil {
					t.Fatal(err)
				}
				defer os.Remove(tmpStdin.Name())
				if _, err := tmpStdin.WriteString(tt.useStdin); err != nil {
					t.Fatal(err)
				}
				tmpStdin.Seek(0, 0)
				r = tmpStdin
			} else {
				f, err := os.Open(fname)
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				r = f
			}

			output := captureOutput(func() {
				processFile(r, tt.opts, "test")
			})

			if output != tt.expected {
				t.Errorf("неверный вывод:\nполучено:\n%q\nожидалось:\n%q", output, tt.expected)
			}
		})
	}
}

// TestPrintWithContext изолированно проверяет вывод контекста.
func TestPrintWithContext(t *testing.T) {
	lines := []string{"a", "b", "c", "d", "e"}
	matches := []bool{false, true, false, true, false}

	tests := []struct {
		name     string
		opts     options
		expected string
	}{
		{
			name:     "без контекста",
			opts:     options{},
			expected: "b\nd\n",
		},
		{
			name:     "после 1 (слияние)",
			opts:     options{after: 1},
			expected: "b\nc\nd\ne\n",
		},
		{
			name:     "до 1 (слияние)",
			opts:     options{before: 1},
			expected: "a\nb\nc\nd\n",
		},
		{
			name:     "до и после (слияние)",
			opts:     options{before: 1, after: 1},
			expected: "a\nb\nc\nd\ne\n",
		},
		{
			name:     "с номерами строк",
			opts:     options{lineNum: true},
			expected: "2:b\n4:d\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := captureOutput(func() {
				printWithContext(lines, matches, tt.opts)
			})
			if output != tt.expected {
				t.Errorf("неверный вывод:\nполучено:\n%q\nожидалось:\n%q", output, tt.expected)
			}
		})
	}
}
