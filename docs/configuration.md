# Configuration Package

The `pkg/config` package provides a robust, type-safe, and modular configuration system for Go applications. While it powers the Sargantana framework, it is designed to be used as a standalone library for any Go project.

## Features

### 1. Modular Configuration
Instead of loading a massive struct at once, the configuration is parsed into a map of raw modules. You can then lazily load and validate only the sections you need using generics.

```yaml
# config.yaml
database:
  host: "localhost"
  port: 5432

redis:
  address: "localhost:6379"
```

```go
// Load the raw config map
cfg, err := config.NewConfig("config.yaml")

// Load specific modules with strong typing
dbConfig, err := config.Get[DatabaseConfig](cfg, "database")
redisConfig, err := config.Get[RedisConfig](cfg, "redis")
```

### 2. Type Safety & Validation
Every configuration struct must implement the `Validatable` interface. This ensures that your configuration is always valid after loading.

```go
type DatabaseConfig struct {
    Host string `yaml:"host"`
    Port int    `yaml:"port"`
}

func (c *DatabaseConfig) Validate() error {
    if c.Port == 0 {
        return errors.New("port is required")
    }
    return nil
}
```

### 3. Secret Injection & Expansion
The package integrates seamlessly with `pkg/config/secrets` to resolve secrets at runtime using `${prefix:key}` syntax.

```yaml
database:
  password: "${vault:db-password}"  # Resolves from Vault
  api_key: "${env:API_KEY}"         # Resolves from Environment
```

### 4. Client Factory Pattern
The `ClientFactory[T]` interface allows configuration structs to directly create configured clients (e.g., database connections), encapsulating the initialization logic.

```go
type ClientFactory[T any] interface {
    Validatable
    CreateClient() (T, error)
}
```

### 5. Multi-Format Support
Supports YAML, JSON, TOML, and XML. Defaults to YAML but can be changed globally.

```go
config.UseFormat(config.JsonFormat)
```

## Standalone Usage

You can use `pkg/config` in any Go application without importing the rest of the Sargantana framework.

### Example

```go
package main

import (
    "fmt"
    "log"
    "github.com/animalet/sargantana-go/pkg/config"
    "github.com/animalet/sargantana-go/pkg/config/secrets"
)

// Define your config struct
type AppConfig struct {
    Name string `yaml:"name"`
    Port int    `yaml:"port"`
}

// Implement Validatable
func (c *AppConfig) Validate() error {
    if c.Port <= 0 {
        return fmt.Errorf("invalid port: %d", c.Port)
    }
    return nil
}

func main() {
    // 1. Register secret providers (optional)
    secrets.Register("env", secrets.NewEnvLoader())

    // 2. Load configuration file
    cfg, err := config.NewConfig("config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // 3. Get and validate specific section
    appConfig, err := config.Get[AppConfig](cfg, "app")
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Starting %s on port %d\n", appConfig.Name, appConfig.Port)
}
```

## Comparison with Viper

Viper is the most popular configuration framework for Go. Here is how Sargantana's `pkg/config` compares:

| Feature | Viper | Sargantana `pkg/config` |
|---------|-------|-------------------------|
| **Philosophy** | "Configuration as a service", global state, dynamic map access. | **Type-safe**, **Stateless**, **Modular**, **Explicit**. |
| **Type Safety** | Loose. Uses `map[string]interface{}` internally. Unmarshaling is a secondary step. | **Strict**. Configuration is always unmarshaled into typed structs immediately. |
| **Validation** | No built-in validation interface. Requires external libraries. | **Built-in `Validatable` interface**. Validation is mandatory and automatic on load. |
| **Secret Management** | Basic env var substitution. | **Extensible `SecretLoader` system**. Supports Vault, AWS, Files, and custom providers with prefix syntax (`${vault:key}`). |
| **Global State** | Heavily relies on global state (`viper.Get`). | **No global state**. Config objects are passed explicitly. |
| **Dependencies** | Heavy dependency tree. | **Lightweight**. Minimal dependencies. |
| **Usage** | Great for CLI tools with many flags and complex overrides. | **Ideal for long-running services** where type safety and validation are critical. |

### Why choose Sargantana Config?

- **Compile-time safety**: You work with Go structs, not string keys.
- **Fail-fast**: Validation happens at load time. If the config is invalid, the app doesn't start.
- **Clean Architecture**: Encourages dependency injection and separation of concerns via the `ClientFactory` pattern.
- **Security**: First-class support for external secret stores like Vault and AWS Secrets Manager.
