package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"
)

func TestRun_SocketToOutput(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	input := strings.NewReader("")
	output := &bytes.Buffer{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	done := make(chan struct{})

	go func() {
		_, _ = server.Write([]byte("hello\n"))
		server.Close()
		close(done)
	}()

	err := run(ctx, client, input, output)

	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}

	<-done

	if !strings.Contains(output.String(), "hello") {
		t.Errorf("expected output to contain 'hello', got %q", output.String())
	}
}

func TestRun_InputToSocket(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	input := strings.NewReader("ping\n")
	output := &bytes.Buffer{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	done := make(chan string, 1)

	go func() {
		buf := make([]byte, 1024)
		n, _ := server.Read(buf)
		done <- string(buf[:n])
		server.Close()
	}()

	err := run(ctx, client, input, output)

	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case received := <-done:
		if received != "ping\n" {
			t.Errorf("expected %q, got %q", "ping\n", received)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for server read")
	}
}

func TestRun_ConnectionClose(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	input := strings.NewReader("")
	output := &bytes.Buffer{}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	go func() {
		server.Close()
	}()

	err := run(ctx, client, input, output)

	if err != nil && err != io.EOF {
		t.Fatalf("unexpected error: %v", err)
	}
}
