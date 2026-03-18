package main

import (
	"reflect"
	"testing"
)

func TestParseFields(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    map[int]bool
		wantErr bool
	}{
		{
			name:    "одиночное поле",
			input:   "2",
			want:    map[int]bool{2: true},
			wantErr: false,
		},
		{
			name:    "несколько полей через запятую",
			input:   "1,3,5",
			want:    map[int]bool{1: true, 3: true, 5: true},
			wantErr: false,
		},
		{
			name:    "диапазон",
			input:   "3-5",
			want:    map[int]bool{3: true, 4: true, 5: true},
			wantErr: false,
		},
		{
			name:    "комбинация одиночных и диапазона",
			input:   "1,3-5,7",
			want:    map[int]bool{1: true, 3: true, 4: true, 5: true, 7: true},
			wantErr: false,
		},
		{
			name:    "диапазон с одинаковыми границами",
			input:   "2-2",
			want:    map[int]bool{2: true},
			wantErr: false,
		},
		{
			name:    "некорректный диапазон (перепутаны границы)",
			input:   "5-3",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "некорректный номер поля (буква)",
			input:   "a",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "пустая строка (не должна быть, но проверим)",
			input:   "",
			want:    map[int]bool{}, // пустая карта
			wantErr: false,
		},
		{
			name:    "диапазон с отрицательным числом",
			input:   "-1-3",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "поле 0 (недопустимо)",
			input:   "0",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseFields(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFields() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseFields() = %v, want %v", got, tt.want)
			}
		})
	}
}
