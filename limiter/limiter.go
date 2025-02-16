package limiter

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type LimitListener struct {
	net.Listener
	perConnLimit      *rate.Limiter
	globalLimit       *rate.Limiter
	activeConnections map[*limitConn]struct{}
	mu                *sync.Mutex
}

func NewLimitListener(l net.Listener, perConnLimit, globalLimit int) *LimitListener {
	return &LimitListener{
		Listener:          l,
		perConnLimit:      rate.NewLimiter(rate.Limit(perConnLimit), perConnLimit),
		globalLimit:       rate.NewLimiter(rate.Limit(globalLimit), globalLimit),
		mu:                &sync.Mutex{},
		activeConnections: make(map[*limitConn]struct{}),
	}
}

func (l *LimitListener) SetLimit(perConnLimit, globalLimit int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.globalLimit.SetLimit(rate.Limit(globalLimit))
	l.globalLimit.SetBurst(globalLimit)
	l.perConnLimit.SetLimit(rate.Limit(perConnLimit))
	l.perConnLimit.SetBurst(perConnLimit)

	for conn := range l.activeConnections {
		conn.perConnLimit.SetLimit(rate.Limit(perConnLimit))
		conn.perConnLimit.SetBurst(perConnLimit)
	}
}

func (l *LimitListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	tcpConn, ok := c.(*net.TCPConn)
	if !ok {
		return nil, errors.New("only TCP connections are supported")
	}

	if err = tcpConn.SetKeepAlive(true); err != nil {
		return nil, err
	}
	if err = tcpConn.SetKeepAlivePeriod(5 * time.Second); err != nil {
		return nil, err
	}

	tc := &limitConn{
		Conn:         tcpConn,
		globalLimit:  l.globalLimit,
		perConnLimit: rate.NewLimiter(l.perConnLimit.Limit(), l.perConnLimit.Burst()),
	}

	l.activeConnections[tc] = struct{}{}
	tc.removeFn = func() {
		l.removeConnection(tc)
	}

	return tc, nil
}

// Remove connection from activeConnections
func (l *LimitListener) removeConnection(tc *limitConn) {
	l.mu.Lock()
	delete(l.activeConnections, tc)
	l.mu.Unlock()
	fmt.Println("Connection removed from activeConnections")
}

type limitConn struct {
	net.Conn
	globalLimit  *rate.Limiter
	perConnLimit *rate.Limiter
	removeFn     func()
}

func (c *limitConn) Close() error {
	c.removeFn()
	return c.Conn.Close()
}

func (c *limitConn) Read(b []byte) (int, error) {
	ctx := context.Background()
	n := len(b)
	if n > 0 {
		if err := c.globalLimit.WaitN(ctx, n); err != nil {
			return 0, err
		}

		if err := c.perConnLimit.WaitN(ctx, n); err != nil {
			return 0, err
		}
	}

	return c.Conn.Read(b)
}

func (c *limitConn) Write(b []byte) (int, error) {
	ctx := context.Background()
	n := len(b)
	if err := c.globalLimit.WaitN(ctx, n); err != nil {
		return 0, err
	}

	if err := c.perConnLimit.WaitN(ctx, n); err != nil {
		return 0, err
	}

	return c.Conn.Write(b)
}

func (c *limitConn) GetLimits() (rate.Limit, rate.Limit) {
	return c.perConnLimit.Limit(), c.globalLimit.Limit()
}
