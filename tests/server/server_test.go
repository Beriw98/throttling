package server

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	t.Run("1 client, 30s, limits (3,10)", func(t *testing.T) {
		m := &metrics{}
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			start(ctx, 3*1024, 10*1024, false, m)
		}()
		time.Sleep(1 * time.Second)

		var wg sync.WaitGroup
		wg.Add(1)

		for range 1 {
			go func() {
				defer wg.Done()
				client()
			}()
		}

		wg.Wait()

		totalDataProcessedKB := m.totalDataProcessed / 1024
		elapsedSeconds := time.Since(m.startTime).Seconds()

		fmt.Println("Total data processed:", totalDataProcessedKB)
		fmt.Println("Elapsed time:", elapsedSeconds)

		// Calculate average bandwidth (in kB/s)
		averageBandwidth := float64(totalDataProcessedKB) / elapsedSeconds

		expectedBandwidth := 3.0
		lowerBound := expectedBandwidth * 0.95
		upperBound := expectedBandwidth * 1.05

		if averageBandwidth < lowerBound || averageBandwidth > upperBound {
			t.Errorf("Average bandwidth out of bounds. Expected: %f, got: %f", expectedBandwidth, averageBandwidth)
		} else {
			fmt.Println("Average bandwidth:", averageBandwidth)
		}

		cancel()
		time.Sleep(2 * time.Second)
	})

	t.Run("2 clients, 30s, limits (6,10)", func(t *testing.T) {
		m := &metrics{}
		ctx, cancel := context.WithCancel(context.Background())

		go func() {
			start(ctx, 6*1024, 10*1024, false, m)
		}()
		time.Sleep(1 * time.Second)

		var wg sync.WaitGroup
		wg.Add(2)

		for range 2 {
			go func() {
				defer wg.Done()
				client()
			}()
		}

		wg.Wait()

		totalDataProcessedKB := m.totalDataProcessed / 1024
		elapsedSeconds := time.Since(m.startTime).Seconds()

		fmt.Println("Total data processed:", totalDataProcessedKB)
		fmt.Println("Elapsed time:", elapsedSeconds)

		// Calculate average bandwidth (in kB/s)
		averageBandwidth := float64(totalDataProcessedKB) / elapsedSeconds / 2

		expectedBandwidth := 5.0
		lowerBound := expectedBandwidth * 0.95
		upperBound := expectedBandwidth * 1.05

		if averageBandwidth < lowerBound || averageBandwidth > upperBound {
			t.Errorf("Average bandwidth out of bounds. Expected: %f, got: %f", expectedBandwidth, averageBandwidth)
		} else {
			fmt.Println("Average bandwidth:", averageBandwidth)
		}

		cancel()
		time.Sleep(2 * time.Second)
	})

	t.Run("2 clients, 30s, limits (3,10), with update after 15s to (6,10)", func(t *testing.T) {
		m := &metrics{}
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			start(ctx, 3*1024, 10*1024, true, m)
		}()

		time.Sleep(1 * time.Second)

		var wg sync.WaitGroup
		wg.Add(2)

		for range 2 {
			go func() {
				defer wg.Done()
				client()
			}()
		}

		wg.Wait()

		elapsedSeconds := time.Since(m.startTime).Seconds()
		total := m.totalDataProcessed

		fmt.Println("Total data processed:", total)
		fmt.Println("Elapsed time:", elapsedSeconds)

		elapsedSecondsAfterLimit := time.Since(m.limitTime).Seconds()

		fmt.Println("Data processed after limit:", m.totalDataProcessedAfterLimit)
		fmt.Println("Elapsed time after limit:", elapsedSecondsAfterLimit)
		// Calculate average bandwidth (in kB/s)
		averageBandwidthBeforeLimit := float64(total-m.totalDataProcessedAfterLimit) / (elapsedSeconds - elapsedSecondsAfterLimit) / 2
		averageBandwidthAfterLimit := float64(m.totalDataProcessedAfterLimit) / elapsedSecondsAfterLimit / 2

		fmt.Println("Average bandwidth before limit:", averageBandwidthBeforeLimit/1024)
		fmt.Println("Average bandwidth after limit:", averageBandwidthAfterLimit/1024)

		averageBandwidth := (float64(total) / elapsedSeconds) / 2 / 1024
		// 3 KB/s for the first ~15 seconds, 5 KB/s for the next ~15 seconds
		expectedBandwidth := 4.0

		// Allow for a 10% margin of error because of the time it takes to update the limits
		lowerBound := expectedBandwidth * 0.90
		upperBound := expectedBandwidth * 1.10

		if averageBandwidth < lowerBound || averageBandwidth > upperBound {
			t.Errorf("Average bandwidth out of bounds. Expected: %f, got: %f", expectedBandwidth, averageBandwidth)
		} else {
			fmt.Println("Average bandwidth:", averageBandwidth)
		}

		cancel()
	})
}

func client() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	data := make([]byte, 1024)

	now := time.Now()
	for time.Since(now) < 30*time.Second {
		_, err := conn.Write(data)
		if err != nil {
			fmt.Println("Error sending data:", err)
			return
		}
	}
}
