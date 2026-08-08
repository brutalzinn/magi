// Package config loads magi.yaml, the single source of truth for models,
// addresses, consensus rounds, and tool execution policy.
package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config mirrors magi.yaml.
type Config struct {
	Runtime   RuntimeConfig          `yaml:"runtime"`
	Consensus ConsensusConfig        `yaml:"consensus"`
	Brains    map[string]BrainConfig `yaml:"brains"`
	Gateway   GatewayConfig          `yaml:"gateway"`
	Tools     ToolsConfig            `yaml:"tools"`
}

type RuntimeConfig struct {
	OllamaURL      string `yaml:"ollama_url"`
	NumCtx         int    `yaml:"num_ctx"`
	KeepAlive      string `yaml:"keep_alive"`
	RequestTimeout string `yaml:"request_timeout"`
}

type ConsensusConfig struct {
	MaxRounds int `yaml:"max_rounds"`
}

type BrainConfig struct {
	Model    string `yaml:"model"`
	GRPCAddr string `yaml:"grpc_addr"`
}

type GatewayConfig struct {
	HTTPAddr string `yaml:"http_addr"`
}

type ToolsConfig struct {
	Expose     []string     `yaml:"expose"`
	Allowlist  []ToolPolicy `yaml:"allowlist"`
	MCPServers []MCPServer  `yaml:"mcp_servers"`
}

// ToolPolicy sets a tool's execution mode: "auto" or "confirm".
type ToolPolicy struct {
	Name string `yaml:"name"`
	Mode string `yaml:"mode"`
}

type MCPServer struct {
	Name    string   `yaml:"name"`
	Command string   `yaml:"command"`
	Args    []string `yaml:"args"`
}

// Load reads and parses a magi.yaml file.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := c.validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

func (c *Config) validate() error {
	if c.Runtime.OllamaURL == "" {
		return fmt.Errorf("runtime.ollama_url is required")
	}
	if c.Consensus.MaxRounds < 1 {
		c.Consensus.MaxRounds = 1
	}
	if len(c.Brains) == 0 {
		return fmt.Errorf("at least one brain must be configured")
	}
	return nil
}

// Brain returns a named brain's config.
func (c *Config) Brain(name string) (BrainConfig, error) {
	b, ok := c.Brains[name]
	if !ok {
		return BrainConfig{}, fmt.Errorf("brain %q not found in config", name)
	}
	return b, nil
}

// RequestTimeoutDuration parses runtime.request_timeout (default 120s).
func (c *Config) RequestTimeoutDuration() time.Duration {
	if c.Runtime.RequestTimeout == "" {
		return 120 * time.Second
	}
	d, err := time.ParseDuration(c.Runtime.RequestTimeout)
	if err != nil {
		return 120 * time.Second
	}
	return d
}
