# Testing Guide

This guide covers how to run tests locally and understand the CI testing pipeline.

## Local Testing

The easiest way to run tests locally is using `make` and `docker-compose`.

### Prerequisites

1.  **Docker & Docker Compose**: Required for integration tests (Redis, Postgres, Mongo, etc.).
2.  **Go 1.25+**: To run the tests.
3.  **Make**: To use the provided convenience commands.

### Running Tests

1.  **Start Test Services**:
    Start the required infrastructure (Redis, Postgres, Vault, etc.) in the background:
    ```bash
    docker-compose up -d
    ```

2.  **Run All Tests**:
    Run unit and integration tests:
    ```bash
    make test
    ```

3.  **Run with Coverage**:
    Run tests and generate a coverage report:
    ```bash
    make test-with-coverage
    ```

4.  **Stop Services**:
    When finished, stop the containers:
    ```bash
    docker-compose down
    ```

### Test Services

The `docker-compose.yml` file provides the following services for integration testing:

*   **Redis**: For session storage and caching tests.
*   **PostgreSQL**: For database integration tests.
*   **MongoDB**: For NoSQL database tests.
*   **Memcached**: For caching tests.
*   **Vault**: For secret management tests.
*   **LocalStack**: Simulates AWS Secrets Manager.
*   **Mock OAuth Server**: Simulates OAuth2 providers for authentication tests.

## Continuous Integration (CI)

This project uses GitHub Actions for Continuous Integration. The pipeline ensures code quality and stability for every push and pull request.

### CI Workflow

The CI pipeline performs the following checks:

1.  **Linting**: Runs `golangci-lint` and `go vet` to catch code style and correctness issues.
2.  **Testing**: Runs all tests with race detection enabled.
3.  **Coverage**: Checks if code coverage meets the required thresholds.
4.  **Security**: Runs `gosec` to scan for security vulnerabilities.

### Running CI Locally

You can simulate the entire CI pipeline locally using the `make ci` command. This is highly recommended before pushing changes.

```bash
make ci
```

This command will:
1.  Configure your environment.
2.  Lint the code.
3.  Run tests with coverage.
4.  Check coverage thresholds.
5.  Run security scans.

**Note**: You must have `docker-compose up -d` running for the tests to pass.
