package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// Тесты для tokenize
func TestTokenize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{"simple", "echo hello world", []string{"echo", "hello", "world"}},
		{"quotes", `echo "hello world"`, []string{"echo", `"hello world"`}},
		{"pipeline", "ps aux | grep go", []string{"ps", "aux", "|", "grep", "go"}},
		{"redirections", "cat < in.txt > out.txt", []string{"cat", "<", "in.txt", ">", "out.txt"}},
		{"and", "echo a && echo b", []string{"echo", "a", "&&", "echo", "b"}},
		{"or", "false || echo ok", []string{"false", "||", "echo", "ok"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.input)
			if !equalSlices(got, tt.want) {
				t.Errorf("tokenize(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// Тесты для expandEnv
func TestExpandEnv(t *testing.T) {
	os.Setenv("TEST_VAR", "world")
	defer os.Unsetenv("TEST_VAR")

	tests := []struct {
		name   string
		tokens []string
		want   []string
	}{
		{"simple", []string{"echo", "$TEST_VAR"}, []string{"echo", "world"}},
		{"quoted", []string{"echo", `"hello $TEST_VAR"`}, []string{"echo", "hello world"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := expandEnv(tt.tokens)
			if !equalSlices(got, tt.want) {
				t.Errorf("expandEnv(%v) = %v, want %v", tt.tokens, got, tt.want)
			}
		})
	}
}

// Тесты для parsePipeline
func TestParsePipeline(t *testing.T) {
	tests := []struct {
		name         string
		tokens       []string
		wantCommands [][]Command
		wantOp       string
	}{
		{
			name:   "single",
			tokens: []string{"ls", "-l"},
			wantCommands: [][]Command{
				{{Args: []string{"ls", "-l"}}},
			},
			wantOp: "",
		},
		{
			name:   "pipeline",
			tokens: []string{"ls", "-l", "|", "grep", "main"},
			wantCommands: [][]Command{
				{
					{Args: []string{"ls", "-l"}},
					{Args: []string{"grep", "main"}},
				},
			},
			wantOp: "",
		},
		{
			name:   "redirections",
			tokens: []string{"cat", "<", "in.txt", ">", "out.txt"},
			wantCommands: [][]Command{
				{{Args: []string{"cat"}, Stdin: "in.txt", Stdout: "out.txt", Append: false}},
			},
			wantOp: "",
		},
		{
			name:   "append",
			tokens: []string{"echo", "hello", ">>", "log.txt"},
			wantCommands: [][]Command{
				{{Args: []string{"echo", "hello"}, Stdout: "log.txt", Append: true}},
			},
			wantOp: "",
		},
		{
			name:   "and",
			tokens: []string{"true", "&&", "echo", "ok"},
			wantCommands: [][]Command{
				{{Args: []string{"true"}}},
				{{Args: []string{"echo", "ok"}}},
			},
			wantOp: "&&",
		},
		{
			name:   "or",
			tokens: []string{"false", "||", "echo", "ok"},
			wantCommands: [][]Command{
				{{Args: []string{"false"}}},
				{{Args: []string{"echo", "ok"}}},
			},
			wantOp: "||",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCommands, gotOp := parsePipeline(tt.tokens)
			if gotOp != tt.wantOp {
				t.Errorf("parsePipeline() op = %q, want %q", gotOp, tt.wantOp)
			}
			if len(gotCommands) != len(tt.wantCommands) {
				t.Errorf("len = %d, want %d", len(gotCommands), len(tt.wantCommands))
				return
			}
			for i := range gotCommands {
				if len(gotCommands[i]) != len(tt.wantCommands[i]) {
					t.Errorf("pipeline[%d] length = %d, want %d", i, len(gotCommands[i]), len(tt.wantCommands[i]))
					continue
				}
				for j := range gotCommands[i] {
					if !equalCommands(gotCommands[i][j], tt.wantCommands[i][j]) {
						t.Errorf("pipeline[%d][%d] = %+v, want %+v", i, j, gotCommands[i][j], tt.wantCommands[i][j])
					}
				}
			}
		})
	}
}

// Тесты для runCommand (без append, т.к. на Windows возникают проблемы)
func TestRunCommand(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "shell_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("output redirection", func(t *testing.T) {
		outFile := filepath.Join(tmpDir, "out.txt")
		cmd := Command{
			Args:   []string{"sh", "-c", `printf "hello\n"`},
			Stdout: outFile,
		}
		code, err := runCommand(cmd)
		if err != nil || code != 0 {
			t.Fatalf("runCommand failed: code=%d err=%v", code, err)
		}
		data, _ := ioutil.ReadFile(outFile)
		if string(data) != "hello\n" {
			t.Errorf("file content = %q, want %q", data, "hello\n")
		}
	})

	t.Run("input redirection", func(t *testing.T) {
		inFile := filepath.Join(tmpDir, "in.txt")
		if err := ioutil.WriteFile(inFile, []byte("content\n"), 0644); err != nil {
			t.Fatal(err)
		}
		outFile := filepath.Join(tmpDir, "out2.txt")
		cmd := Command{
			Args:   []string{"cat"},
			Stdin:  inFile,
			Stdout: outFile,
		}
		code, err := runCommand(cmd)
		if err != nil || code != 0 {
			t.Fatalf("runCommand failed: code=%d err=%v", code, err)
		}
		data, _ := ioutil.ReadFile(outFile)
		if string(data) != "content\n" {
			t.Errorf("file content = %q, want %q", data, "content\n")
		}
	})
}

// Тесты для runPipeline
func TestRunPipeline(t *testing.T) {
	t.Run("single command", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmds := []Command{
			{Args: []string{"sh", "-c", `printf "hello\n"`}},
		}
		code, err := runPipeline(cmds)
		w.Close()
		os.Stdout = oldStdout

		if err != nil || code != 0 {
			t.Errorf("runPipeline failed: code=%d err=%v", code, err)
		}
		out, _ := ioutil.ReadAll(r)
		if string(out) != "hello\n" {
			t.Errorf("output = %q, want %q", string(out), "hello\n")
		}
	})

	t.Run("pipeline", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmds := []Command{
			{Args: []string{"sh", "-c", `printf "hello world\n"`}},
			{Args: []string{"wc", "-w"}},
		}
		code, err := runPipeline(cmds)
		w.Close()
		os.Stdout = oldStdout

		if err != nil || code != 0 {
			t.Errorf("runPipeline failed: code=%d err=%v", code, err)
		}
		out, _ := ioutil.ReadAll(r)
		if string(out) != "2\n" {
			t.Errorf("output = %q, want %q", string(out), "2\n")
		}
	})

	t.Run("pipeline with grep", func(t *testing.T) {
		tmpDir, err := ioutil.TempDir("", "pipeline_test")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(tmpDir)
		dataFile := filepath.Join(tmpDir, "data.txt")
		if err := ioutil.WriteFile(dataFile, []byte("line1\nline2\n"), 0644); err != nil {
			t.Fatal(err)
		}

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmds := []Command{
			{Args: []string{"cat", dataFile}},
			{Args: []string{"grep", "line2"}},
		}
		code, err := runPipeline(cmds)
		w.Close()
		os.Stdout = oldStdout

		if err != nil || code != 0 {
			t.Errorf("runPipeline failed: code=%d err=%v", code, err)
		}
		out, _ := ioutil.ReadAll(r)
		if string(out) != "line2\n" {
			t.Errorf("output = %q, want %q", string(out), "line2\n")
		}
	})
}

// Вспомогательные функции
func equalSlices(a, b []string) bool {
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

func equalCommands(a, b Command) bool {
	if !equalSlices(a.Args, b.Args) {
		return false
	}
	if a.Stdin != b.Stdin || a.Stdout != b.Stdout || a.Append != b.Append {
		return false
	}
	return true
}
