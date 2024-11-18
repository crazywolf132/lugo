package lugo

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Environment represents a configuration environment (e.g., dev, staging, prod)
type Environment struct {
	// Name of the environment
	Name string
	// Base configuration that applies to all environments
	BaseConfig string
	// Environment-specific configuration
	EnvConfig string
	// Environment variables prefix for this environment
	EnvPrefix string
	// Additional configuration paths
	IncludePaths []string
}

// EnvManager manages different configuration environments
type EnvManager struct {
	cfg          *Config
	environments map[string]*Environment
	activeEnv    string
	configDir    string
}

// NewEnvManager creates a new environment manager
func (c *Config) NewEnvManager(configDir string) *EnvManager {
	return &EnvManager{
		cfg:          c,
		environments: make(map[string]*Environment),
		configDir:    configDir,
	}
}

// RegisterEnvironment registers a new environment
func (em *EnvManager) RegisterEnvironment(env *Environment) error {
	if env.Name == "" {
		return fmt.Errorf("environment name cannot be empty")
	}

	// Validate base config exists
	if env.BaseConfig != "" {
		basePath := filepath.Join(em.configDir, env.BaseConfig)
		if _, err := os.Stat(basePath); err != nil {
			return fmt.Errorf("base config not found: %s", basePath)
		}
	}

	// Validate env config exists
	if env.EnvConfig != "" {
		envPath := filepath.Join(em.configDir, env.EnvConfig)
		if _, err := os.Stat(envPath); err != nil {
			return fmt.Errorf("environment config not found: %s", envPath)
		}
	}

	em.environments[env.Name] = env
	return nil
}

// ActivateEnvironment activates a specific environment
func (em *EnvManager) ActivateEnvironment(name string) error {
	env, ok := em.environments[name]
	if !ok {
		return fmt.Errorf("environment not found: %s", name)
	}

	// Load base configuration first
	if env.BaseConfig != "" {
		basePath := filepath.Join(em.configDir, env.BaseConfig)
		if err := em.cfg.LoadFile(nil, basePath); err != nil {
			return fmt.Errorf("failed to load base config: %w", err)
		}
	}

	// Load environment-specific configuration
	if env.EnvConfig != "" {
		envPath := filepath.Join(em.configDir, env.EnvConfig)
		if err := em.cfg.LoadFile(nil, envPath); err != nil {
			return fmt.Errorf("failed to load environment config: %w", err)
		}
	}

	// Load additional include paths
	for _, includePath := range env.IncludePaths {
		path := filepath.Join(em.configDir, includePath)
		if err := em.cfg.LoadFile(nil, path); err != nil {
			return fmt.Errorf("failed to load include file %s: %w", includePath, err)
		}
	}

	// Load environment variables with prefix
	if env.EnvPrefix != "" {
		if err := em.loadEnvVars(env.EnvPrefix); err != nil {
			return fmt.Errorf("failed to load environment variables: %w", err)
		}
	}

	em.activeEnv = name
	return nil
}

// GetActiveEnvironment returns the currently active environment
func (em *EnvManager) GetActiveEnvironment() string {
	return em.activeEnv
}

// loadEnvVars loads environment variables with the specified prefix into Lua globals
func (em *EnvManager) loadEnvVars(prefix string) error {
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key, value := parts[0], parts[1]
		if !strings.HasPrefix(key, prefix) {
			continue
		}

		// Convert environment variable name to Lua variable name
		luaName := strings.ToLower(strings.TrimPrefix(key, prefix))
		luaName = strings.ReplaceAll(luaName, "_", ".")

		// Set as global in Lua
		if err := em.cfg.SetGlobal(luaName, value); err != nil {
			return fmt.Errorf("failed to set environment variable %s: %w", key, err)
		}
	}

	return nil
}
