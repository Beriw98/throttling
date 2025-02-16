# Throttling

This project implements a simple TCP limiter with rate-limited connections. It can  enforce connection limits both per-connection and globally across all connections. It also supports dynamic updates to the rate limits during runtime.

## Project Structure
```
├── /limiter
│   └── limiter.go # Implements the rate-limiting functionality for TCP connections.
├── /tests
│   └──/server
│       ├── server.go # Implements the server logic with connection handling and rate-limiting.
│       └── server_test.go # Contains test cases to verify the server and throttling behavior.
```

## Testing
To run the tests, execute the following command:
```bash
go test ./tests/server
```
They will verify the server logic and the rate-limiting functionality.
They should take around 1.5 minutes to complete.

### Test Cases
1. Test the server with a single connection and per-connection limit.
2. Test the server with multiple connections and global limit.
3. Test the server with multiple connections and rate limiting change in runtime.

## Potential Improvements
1. Remove connections from the rate limiter when they are closed unexpectedly, to free up resources.
