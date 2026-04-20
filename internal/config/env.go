package config

import (
	flags "FoPQer/go-fermart/internal/config/flag"
	"net/url"
	"os"
)

type Config struct {
	SecretKey []byte
	RunAddr string
	DatabaseURI string
	AccrualAddress string
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) Load() {
	c.SecretKey = []byte(os.Getenv("SECRET_KEY"))
	c.RunAddr = os.Getenv("RUN_ADDRESS")
	c.DatabaseURI = os.Getenv("DATABASE_URI")
	c.AccrualAddress = os.Getenv("ACCRUAL_SYSTEM_ADDRESS")
}

func (c *Config) GetSecretKey() []byte {
	return c.SecretKey
}

func (c *Config) GetRunAddr() string {
	if c.RunAddr != "" {
		return url.PathEscape(c.RunAddr)
	} else {
		return url.PathEscape(flags.GetFlagRunAddr())
	}
}

func (c *Config) GetDatabaseURI() string {
	if c.DatabaseURI != "" {
		return c.DatabaseURI
	} else {
		return flags.GetFlagDatabaseURI()
	}
}

func (c *Config) GetAccrualAddress() string {
	if c.AccrualAddress != "" {
		return c.AccrualAddress
	} else {
		return flags.GetFlagAccrualAddress()
	}
}