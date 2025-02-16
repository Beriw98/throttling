package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"

	"throttling/limiter"
)

type metrics struct {
	totalDataProcessed           uint64
	totalDataProcessedAfterLimit uint64
	startTime                    time.Time
	limitTime                    time.Time
}

func start(ctx context.Context, per, global int, withUpdate bool, metrics *metrics) {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		panic(err)
	}
	l := limiter.NewLimitListener(listener, per, global)
	fmt.Println("Server started on port 8080")

	defer listener.Close()

	// Use sync.Once to ensure SetLimits is called only once
	var once sync.Once

	var serverWg sync.WaitGroup
	serverWg.Add(1)
	go func() {
		defer serverWg.Done()
		for {
			conn, err := l.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					// Server is shutting down
					return
				default:
					fmt.Println("Error accepting connection:", err)
					continue
				}
			}

			// Set the limits only once
			once.Do(func() {
				metrics.startTime = time.Now() // start tracking time
				if withUpdate {
					go func() {
						time.Sleep(15 * time.Second)
						fmt.Println("Updating limits...")
						metrics.limitTime = time.Now()
						l.SetLimit(6*1024, 10*1024)
					}()
				}
			})

			serverWg.Add(1)
			go func(conn net.Conn) {
				defer serverWg.Done()
				handleConnection(conn, metrics)
			}(conn)
		}
	}()

	// Wait for the shutdown signal
	<-ctx.Done()
	fmt.Println("Shutting down server...")
	// Close the listener and wait for active connections to finish
	listener.Close()
	serverWg.Wait()
	fmt.Println("Server stopped")
}

func handleConnection(conn net.Conn, metrics *metrics) {
	fmt.Println("Accepted connection")
	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		metrics.totalDataProcessed += uint64(n)
		if metrics.limitTime.After(metrics.startTime) {
			metrics.totalDataProcessedAfterLimit += uint64(n)
		}
	}
}
