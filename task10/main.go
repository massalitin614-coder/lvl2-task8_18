package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

func main() {
	timeout := flag.Duration("timeout", 10*time.Second, "connection timeout")
	flag.Parse()

	if flag.NArg() < 2 {
		fmt.Println("usage: go-telnet [--timeout=10s] host port")
		os.Exit(1)
	}

	host := flag.Arg(0)
	port := flag.Arg(1)
	address := net.JoinHostPort(host, port)

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("connection timeout")
		} else {
			fmt.Println("connection error:", err)
		}
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("Connected to", address)

	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	err = run(ctx, conn, os.Stdin, os.Stdout)
	if err != nil && err != context.Canceled && err != io.EOF {
		fmt.Println("\nError:", err)
	} else {
		fmt.Println("\nConnection closed")
	}
}

func run(ctx context.Context, conn net.Conn, in io.Reader, out io.Writer) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	// socket -> stdout
	wg.Add(1)
	go func() {
		defer wg.Done()

		buf := make([]byte, 1024)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				errCh <- err
				return
			}

			if _, err := out.Write(buf[:n]); err != nil {
				errCh <- err
				return
			}
		}
	}()

	// stdin -> socket
	wg.Add(1)
	go func() {
		defer wg.Done()

		reader := bufio.NewReader(in)

		for {
			data, err := reader.ReadBytes('\n')
			if err != nil {
				errCh <- err
				return
			}

			if _, err := conn.Write(data); err != nil {
				errCh <- err
				return
			}
		}
	}()

	var err error
	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	}

	cancel()
	wg.Wait()

	return err
}
