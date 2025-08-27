# Testing Guide

This guide covers how to run tests locally, set up the testing environment, and understand the different types of tests in Sargantana Go.

## Prerequisites

- Go 1.25 or later
- Docker and Docker Compose
- Make

## Testing Infrastructure

Some tests require external services like databases and authentication providers. A `docker-compose.yml` file is provided to set up the required test infrastructure locally.

### Required Services

The docker-compose setup includes:

- **Neo4j** (port 7687): Graph database for database integration tests
- **Valkey/Redis** (port 6379): In-memory database for session storage tests  
- **Mock OAuth2 Server** (port 8080): Mock authentication provider for auth controller tests

### Starting Test Services

```bash
# Start all test services in the background
docker-compose up -d

# Verify services are running
docker-compose ps

# View service logs if needed
docker-compose logs

# Stop and clean up test services when done
docker-compose down
```

## Running Tests

### Quick Test Commands

```bash
# Run all tests (requires docker-compose services)
make test

# Run tests with coverage report
make test-coverage

# Run all CI checks locally (linting, formatting, tests)
make ci
```

### Manual Test Commands

```bash
# Run tests for specific packages
go test ./controller/...
go test ./server/...
go test ./database/...

# Run tests with verbose output
go test -v ./...

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -cover ./...

# Generate detailed coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Running Specific Tests

```bash
# Run a specific test function
go test -run TestAuthController ./controller/

# Run tests matching a pattern
go test -run "TestAuth.*" ./controller/

# Run tests in a specific file
go test ./controller/ -run TestAuth
```

## Test Categories

### Unit Tests

Unit tests test individual functions and methods in isolation. These tests typically don't require external services and run quickly.

**Location**: `*_test.go` files alongside source code
**Command**: `go test ./... -short`

Examples:
- Configuration parsing tests
- Controller binding tests
- Utility function tests

### Integration Tests

Integration tests verify that different components work together correctly. These tests require external services.

**Requirements**: Docker Compose services must be running
**Command**: `go test ./...` (includes integration tests)

Examples:
- Database connection tests
- Authentication flow tests
- Session management tests

### End-to-End Tests

End-to-end tests verify complete user workflows from start to finish.

**Requirements**: Full application stack running
**Location**: May be in separate test files or external test suites

## Test Services Details

### Neo4j Database

- **Purpose**: Testing database integration features
- **Connection**: `bolt://localhost:7687`
- **Credentials**: `neo4j/testpassword`
- **Health Check**: Automatically verified by docker-compose

```bash
# Connect manually to Neo4j for debugging
docker-compose exec neo4j cypher-shell -u neo4j -p testpassword
```

### Redis/Valkey

- **Purpose**: Testing session management and caching
- **Connection**: `localhost:6379`
- **Configuration**: No authentication required for testing

```bash
# Connect to Redis for debugging
docker-compose exec valkey redis-cli
```

### Mock OAuth2 Server

- **Purpose**: Testing authentication flows without real OAuth providers
- **URL**: `http://localhost:8080`
- **Configuration**: Pre-configured with test tokens and user data

The mock server provides test endpoints for OAuth flows and returns predictable user data for testing.

## Running Tests Without Docker

While the docker-compose setup is recommended for local development, some tests can run without it by using in-memory alternatives or by skipping integration tests.

### Test Categories That Require Docker

- Database integration tests (Neo4j, Redis)
- Authentication controller tests with real OAuth flows
- Session management tests with Redis backend

### Test Categories That Don't Require Docker

- Unit tests for controllers, config, utilities
- In-memory session tests
- Static file serving tests
- Load balancer logic tests

### Skipping Integration Tests

```bash
# Run only unit tests (skip integration tests)
go test -short ./...

# Run tests with custom build tags to skip certain tests
go test -tags=unit ./...
```

## Continuous Integration

The project uses GitHub Actions for CI. The CI pipeline:

1. Sets up Go environment
2. Starts required services with docker-compose
3. Runs linting and formatting checks
4. Executes all tests with coverage reporting
5. Uploads coverage results

### Local CI Simulation

```bash
# Run the same checks as CI
make ci

# Individual CI steps
make lint
make format
make test-coverage
```

## Test Configuration

### Environment Variables for Testing

Some tests may require specific environment variables:

```bash
# For testing authentication providers
export OAUTH_MOCK_SERVER_URL="http://localhost:8080"

# For database tests
export NEO4J_URI="bolt://localhost:7687"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="testpassword"

# For Redis tests
export REDIS_URL="localhost:6379"
```

### Test Data

Test data and fixtures are typically defined within test files or in dedicated test data directories. Look for:

- Test user data in auth tests
- Mock HTTP responses
- Sample configuration files

## Debugging Tests

### Common Issues

1. **Services not running**: Ensure `docker-compose up -d` was executed
2. **Port conflicts**: Check if ports 6379, 7687, or 8080 are already in use
3. **Permission issues**: Ensure Docker has proper permissions

### Debugging Commands

```bash
# Check if services are healthy
docker-compose ps

# View service logs
docker-compose logs neo4j
docker-compose logs valkey
docker-compose logs mock-oauth2-server

# Restart specific service
docker-compose restart neo4j

# Clean up and restart all services
docker-compose down && docker-compose up -d
```

### Test Debugging

```bash
# Run tests with verbose output
go test -v ./controller/

# Run specific test with debug output
go test -v -run TestSpecificFunction ./controller/

# Run tests with race detection
go test -race ./...
```

### Running with debug logs

```bash
# Run the main application in debug mode
go run -race ./main -debug

# Or build and run with debug symbols
go build -o sargantana-go ./main
./sargantana-go -debug
```

## Writing New Tests

### Test Structure

Follow Go testing conventions:

```go
func TestFunctionName(t *testing.T) {
    // Setup
    
    // Test cases
    tests := []struct {
        name     string
        input    string
        expected string
    }{
        {"case 1", "input1", "expected1"},
        {"case 2", "input2", "expected2"},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

### Integration Test Guidelines

1. Check if required services are available
2. Use test-specific configuration
3. Clean up resources after tests
4. Use meaningful test data
5. Test both success and failure scenarios

### Mock vs Real Services

- Use real services (docker-compose) for integration tests
- Use mocks for unit tests
- Consider using test containers for complex scenarios

## Performance Testing

### Benchmarking

```bash
# Run benchmarks
go test -bench=. ./...

# Run specific benchmarks
go test -bench=BenchmarkAuthController ./controller/

# Generate CPU profile during benchmarks
go test -bench=. -cpuprofile=cpu.prof ./...
```

### Load Testing

For load testing the complete application:

```bash
# Install hey (HTTP load testing tool)
go install github.com/rakyll/hey@latest

# Basic load test
hey -n 1000 -c 10 http://localhost:8080/

# Test specific endpoints
hey -n 500 -c 5 http://localhost:8080/auth/user
```

## Troubleshooting

### Common Test Failures

1. **Connection refused errors**: Services not started or not healthy
2. **Authentication test failures**: Mock OAuth server not running
3. **Database test failures**: Neo4j not initialized or connection issues
4. **Session test failures**: Redis not available

### Getting Help

1. Check service logs: `docker-compose logs [service-name]`
2. Verify service health: `docker-compose ps`
3. Check for port conflicts: `netstat -tulpn | grep :8080`
4. Review test output for specific error messages
