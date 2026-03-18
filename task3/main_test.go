package main

import (
	"os"
	"reflect"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name    string   // название теста
		args    []string // входные аргументы (без имени программы)
		want    *Flags   // ожидаемая структура
		wantErr bool     // ожидаем ли ошибку
	}{
		{
			name: "no args",
			args: []string{},
			want: &Flags{keyColumn: 0},
		},
		{
			name: "single flag -n",
			args: []string{"-n"},
			want: &Flags{keyColumn: 0, numeric: true},
		},
		{
			name: "combined flags -nr",
			args: []string{"-nr"},
			want: &Flags{keyColumn: 0, numeric: true, reverse: true},
		},
		{
			name: "flag -k with separate argument",
			args: []string{"-k", "2", "file.txt"},
			want: &Flags{keyColumn: 2, files: []string{"file.txt"}},
		},
		{
			name: "flag -k2 combined",
			args: []string{"-k2", "file.txt"},
			want: &Flags{keyColumn: 2, files: []string{"file.txt"}},
		},
		{
			name: "multiple files and flags",
			args: []string{"-u", "-b", "a.txt", "b.txt"},
			want: &Flags{keyColumn: 0, unique: true, trimSpace: true, files: []string{"a.txt", "b.txt"}},
		},
		{
			name: "double dash",
			args: []string{"--", "-n", "file.txt"},
			want: &Flags{keyColumn: 0, files: []string{"-n", "file.txt"}},
		},
		{
			name:    "unknown flag",
			args:    []string{"-x"},
			wantErr: true,
		},
		{
			name:    "-k without argument",
			args:    []string{"-k"},
			wantErr: true,
		},
		{
			name:    "-k with zero",
			args:    []string{"-k0"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Сохраняем и восстанавливаем os.Args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = append([]string{"sort"}, tt.args...)

			got, err := parseArgs()
			if (err != nil) != tt.wantErr {
				t.Errorf("parseArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			// Сравниваем структуры (можно использовать reflect.DeepEqual)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseArgs() = %v, want %v", got, tt.want)
			}
		})
	}
}
