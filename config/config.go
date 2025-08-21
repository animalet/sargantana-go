package config

type Config struct {
	address           string
	redisSessionStore string
	secretsDir        string
	debug             bool
	sessionName       string
}

func NewConfig(address, redisSessionStore, secretsDir string, debug bool, sessionName string) *Config {
	return &Config{
		address:           address,
		redisSessionStore: redisSessionStore,
		secretsDir:        secretsDir,
		debug:             debug,
		sessionName:       sessionName,
	}
}

func (c *Config) Address() string {
	return c.address
}

func (c *Config) RedisSessionStore() string {
	return c.redisSessionStore
}

func (c *Config) SecretsDir() string {
	return c.secretsDir
}

func (c *Config) Debug() bool {
	return c.debug
}

func (c *Config) SessionName() string {
	return c.sessionName
}
