package config

import "github.com/BurntSushi/toml"

// Mongo is the MongoDB config
type Mongo struct {
	URI      string `toml:"uri"`
	Database string `toml:"database"`
}

// Config is the app config
type Config struct {
	Mongo Mongo
}

// Load receive the config and data and generate/return a new Config
func Load(data []byte) *Config {
	config := Config{}
	toml.Unmarshal(data, &config)
	return &config
}
