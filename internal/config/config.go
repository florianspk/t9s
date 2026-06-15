package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

type TalosConfig struct {
	Context  string             `yaml:"context"`
	Contexts map[string]Context `yaml:"contexts"`
}

type Context struct {
	Endpoints []string `yaml:"endpoints"`
	Nodes     []string `yaml:"nodes"`
	CA        string   `yaml:"ca"`
	Crt       string   `yaml:"crt"`
	Key       string   `yaml:"key"`
}

func Load(path string) (*TalosConfig, error) {
	if path == "" {
		if env := os.Getenv("TALOSCONFIG"); env != "" {
			path = env
		} else {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, fmt.Errorf("home dir: %w", err)
			}
			path = filepath.Join(home, ".talos", "config")
		}
	}

	data, err := os.ReadFile(path) // #nosec G304 G703 -- path is the user-supplied talosconfig flag, intentional
	if err != nil {
		return nil, fmt.Errorf("read talosconfig: %w", err)
	}

	var cfg TalosConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse talosconfig: %w", err)
	}

	return &cfg, nil
}

func (c *TalosConfig) CurrentContext() *Context {
	if ctx, ok := c.Contexts[c.Context]; ok {
		return &ctx
	}
	return nil
}

func (c *TalosConfig) ContextNames() []string {
	names := make([]string, 0, len(c.Contexts))
	for name := range c.Contexts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
