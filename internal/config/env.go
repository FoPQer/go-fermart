package config

import "os"

type Config struct {
	SecretKey []byte
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Load() {
	c.SecretKey = []byte(os.Getenv("SECRET_KEY"))
}

func (c *Config) GetSecretKey() []byte {
	return c.SecretKey
}