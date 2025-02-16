package limiter

import (
	"fmt"
	"golang.org/x/time/rate"
	"net"
	"sync"
	"testing"
	"time"
)

func TestLimitListener_Accept(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()

	perConnLimit := 3
	globalLimit := 10
	limiter := NewLimitListener(listener, perConnLimit, globalLimit)

	// Start accepting connections in a separate goroutine
	go func() {
		_, err := limiter.Accept()
		if err != nil {
			t.Errorf("Error accepting connection: %v", err)
		}
	}()

	// Dial a connection to the server
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Error dialing connection: %v", err)
	}
	defer conn.Close()

	// Give the connection time to be accepted
	time.Sleep(1 * time.Second)
}

func TestLimitListener_SetLimit(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()

	perConnLimit := 3
	globalLimit := 10
	limiter := NewLimitListener(listener, perConnLimit, globalLimit)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := limiter.Accept()
		if err != nil {
			t.Errorf("Error accepting connection: %v", err)
		}
	}()

	// Dial a connection to the server
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Error dialing connection: %v", err)
	}
	defer conn.Close()

	wg.Wait()

	// Change the limits dynamically
	limiter.SetLimit(5, 15)

	if limiter.perConnLimit.Limit() != rate.Limit(5) {
		t.Errorf("Expected per-connection limit 5, got %v", limiter.perConnLimit.Limit())
	}

	if limiter.globalLimit.Limit() != rate.Limit(15) {
		t.Errorf("Expected global limit 15, got %v", limiter.globalLimit.Limit())
	}

	if len(limiter.activeConnections) != 1 {
		t.Errorf("Expected 1 active connection, got %d", len(limiter.activeConnections))
	}

	for lConn := range limiter.activeConnections {
		pcl, gl := lConn.GetLimits()
		if pcl != rate.Limit(5) {
			t.Errorf("Expected per-connection limit 5, got %v", pcl)
		}

		if gl != rate.Limit(15) {
			t.Errorf("Expected global limit 15, got %v", gl)
		}
	}
}

func TestNewLimitListener(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()

	perConnLimit := 3
	globalLimit := 10
	limiter := NewLimitListener(listener, perConnLimit, globalLimit)

	if limiter.perConnLimit.Limit() != rate.Limit(perConnLimit) {
		t.Errorf("Expected per-connection limit %d, got %v", perConnLimit, limiter.perConnLimit.Limit())
	}

	if limiter.globalLimit.Limit() != rate.Limit(globalLimit) {
		t.Errorf("Expected global limit %d, got %v", globalLimit, limiter.globalLimit.Limit())
	}
}

func Test_limitConn_ReadWriteClose(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Error creating listener: %v", err)
	}
	defer listener.Close()

	perConnLimit := 100
	globalLimit := 100
	limiter := NewLimitListener(listener, perConnLimit, globalLimit)

	var wg sync.WaitGroup
	wg.Add(1)
	// Accept a connection in a goroutine
	go func() {
		defer wg.Done()
		conn, err := limiter.Accept()
		if err != nil {
			t.Errorf("Error accepting connection: %v", err)
		}
		limConn := conn.(*limitConn)

		fmt.Println("Connection accepted")
		// Test writing data to the connection
		data := []byte("test data")
		_, err = limConn.Write(data)
		if err != nil {
			t.Errorf("Error writing data: %v", err)
		}

		// Test reading data from the connection
		_, err = limConn.Read(data)
		if err != nil {
			t.Errorf("Error reading data: %v", err)
		}

		// Test closing the connection
		if err = limConn.Close(); err != nil {
			t.Errorf("Error closing connection: %v", err)
		}
	}()

	// Dial a connection to the server
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Error dialing connection: %v", err)
	}

	if _, err = conn.Write([]byte("test data")); err != nil {
		t.Fatalf("Error writing data: %v", err)
	}

	wg.Wait()

	if len(limiter.activeConnections) != 0 {
		t.Errorf("Expected 0 active connections, got %d", len(limiter.activeConnections))
	}
}
