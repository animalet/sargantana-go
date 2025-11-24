# Config Package vs Viper: A Feature Comparison

This document provides a detailed comparison between the `sargantana-go` `config` package and [Viper](https://github.com/spf13/viper), a popular configuration solution for Go applications.

## Feature Comparison Matrix

| Feature | Sargantana Config | Viper |
| :--- | :--- | :--- |
| **Design Philosophy** | Explicit, Type-Safe, Immutable, Modular | "Batteries Included", Dynamic, Mutable, Singleton-heavy |
| **Supported Formats** | YAML, JSON, TOML, XML | YAML, JSON, TOML, HCL, envfile, Java properties |
| **Type Safety** | **High**. Uses Generics (`Get[T]`) and `Validatable` interface. | **Medium**. Relies on `mapstructure` and loose typing before unmarshaling. |
| **Immutability** | **Yes**. Config is loaded once; `Get` returns new instances. | **No**. Config can change at runtime (e.g., via `WatchConfig`). |
| **Environment Variables** | Explicit expansion via `${env:VAR}` syntax. | Automatic binding (`AutomaticEnv`), prefixing, and aliasing. |
| **Secret Management** | **Built-in**. First-class support for secret expansion (`${vault:...}`, `${file:...}`). | **External**. Requires manual handling or plugins/remote providers. |
| **Flag Parsing** | No. | **Yes**. Native integration with `pflag`. |
| **Live Watching** | No. | **Yes**. Can watch config files and reload on change. |
| **Remote Config** | No (except via secret providers). | **Yes**. Etcd, Consul, Firestore, etc. |
| **Dependencies** | Lightweight (`yaml.v3`, `go-toml`, `pkg/errors`). | Heavy (includes `fsnotify`, `mapstructure`, `pflag`, `afero`, etc.). |

## Sargantana Config Analysis

The `sargantana-go` config package is designed for applications that prioritize correctness, stability, and explicit configuration over dynamic behavior.

### Strengths
1.  **Type Safety & Validation**: By enforcing `Validatable` interface and using generics, it ensures that configuration is valid immediately upon access. Errors are caught early.
2.  **Immutability**: The configuration object is immutable after loading. This prevents race conditions and unexpected runtime behavior caused by global state changes.
3.  **Built-in Secret Expansion**: The integration with the `secrets` package allows seamless injection of secrets from Vault, files, or environment variables directly into config fields using `${prefix:key}` syntax.
4.  **Modular Design**: The `ReadModular` approach encourages breaking configuration into logical sections (modules), which promotes better organization in large applications.
5.  **Simplicity**: The API is small and easy to understand. There is no "magic" merging of flags, env vars, and files unless explicitly requested via expansion.

### Weaknesses
1.  **No Live Reloading**: Changing configuration requires a service restart. This is often a feature in production environments (immutable infrastructure), but can be a limitation for some use cases.
2.  **No Flag Integration**: Does not handle command-line flags. This must be handled separately (e.g., using `flag` or `cobra`) and passed to the application.
3.  **Less Flexible Env Binding**: Environment variables must be explicitly referenced in the config file (e.g., `${env:PORT}`). Viper can automatically map `MYAPP_PORT` to `port` without explicit config file entries.

## Viper Analysis

Viper is the de facto standard for Go configuration, offering a massive feature set designed to handle almost any configuration scenario.

### Strengths
1.  **Feature Rich**: Handles files, environment variables, flags, and remote configuration systems out of the box.
2.  **Live Watching**: Can detect file changes and reload configuration at runtime, allowing for dynamic updates without restarts.
3.  **Ecosystem Integration**: Widely used and understood by Go developers. Integrates seamlessly with `Cobra` for CLI applications.
4.  **Hierarchical Merging**: Can merge configuration from multiple sources (defaults < config file < env < flags), allowing for flexible overrides.

### Weaknesses
1.  **Type Safety**: Heavily relies on `map[string]interface{}` internally. Unmarshaling into structs can sometimes be brittle or require specific tags (`mapstructure`).
2.  **Mutability & Global State**: Often used as a global singleton. Mutable state can lead to race conditions if not handled carefully during reloads.
3.  **Complexity & Size**: Brings in a large number of dependencies. The API surface is huge, which can be overwhelming for simple needs.
4.  **Secret Handling**: While it supports remote config, handling sensitive secrets often requires additional setup or custom decoding logic compared to Sargantana's built-in expansion.

## Conclusion

**Choose Sargantana Config if:**
*   You are building a backend service where **stability and correctness** are paramount.
*   You prefer **immutable infrastructure** (restart on config change).
*   You want **strict validation** of configuration at startup.
*   You need first-class support for **secret injection** (Vault, etc.) without extra boilerplate.

**Choose Viper if:**
*   You are building a **CLI tool** (especially with Cobra) that needs complex flag/config merging.
*   You need **live reloading** of configuration.
*   You need to support **remote configuration backends** (Etcd, Consul) natively.
*   You want to allow users to override any config option via environment variables without explicit placeholders.
