package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type config struct {
	JSON    *bool      `toml:"json"`
	All     *bool      `toml:"all"`
	TCP     *bool      `toml:"tcp"`
	UDP     *bool      `toml:"udp"`
	Address *string    `toml:"address"`
	State   []string   `toml:"state"`
	Host    *bool      `toml:"host"`
	Verbose *bool      `toml:"verbose"`
	Name    *string    `toml:"name"`
	Free    freeConfig `toml:"free"`
}

type freeConfig struct {
	DefaultStart *int    `toml:"default_start"`
	DefaultRange *string `toml:"default_range"`
}

func loadConfig(path string, explicit bool) (config, error) {
	var cfg config
	if path == "" {
		path = defaultConfigPath()
	}
	if path == "" {
		return cfg, nil
	}
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) && !explicit {
			return cfg, nil
		}
		return cfg, fmt.Errorf("config file not found: %s", path)
	}
	md, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return cfg, err
	}
	allowed := map[string]bool{
		"json": true, "all": true, "tcp": true, "udp": true, "address": true,
		"state": true, "host": true, "verbose": true, "name": true, "free": true,
		"free.default_start": true, "free.default_range": true,
	}
	for _, key := range md.Keys() {
		k := key.String()
		if !allowed[k] {
			return cfg, fmt.Errorf("unknown config key: %s", k)
		}
	}
	if cfg.Free.DefaultStart != nil && cfg.Free.DefaultRange != nil {
		return cfg, fmt.Errorf("free.default_start and free.default_range are mutually exclusive")
	}
	return cfg, nil
}

func defaultConfigPath() string {
	if runtimeGOOS() == "windows" {
		if appData := os.Getenv("AppData"); appData != "" {
			return filepath.Join(appData, "port-ls", "config.toml")
		}
		return ""
	}
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, "port-ls", "config.toml")
}

var runtimeGOOS = func() string { return runtime.GOOS }

func applyEnv(cfg *config) error {
	boolVars := map[string]**bool{
		"PORT_LS_JSON":    &cfg.JSON,
		"PORT_LS_ALL":     &cfg.All,
		"PORT_LS_TCP":     &cfg.TCP,
		"PORT_LS_UDP":     &cfg.UDP,
		"PORT_LS_HOST":    &cfg.Host,
		"PORT_LS_VERBOSE": &cfg.Verbose,
	}
	for name, target := range boolVars {
		v, ok := os.LookupEnv(name)
		if !ok {
			continue
		}
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		*target = &b
	}
	if v, ok := os.LookupEnv("PORT_LS_ADDRESS"); ok {
		cfg.Address = &v
	}
	if v, ok := os.LookupEnv("PORT_LS_NAME"); ok {
		cfg.Name = &v
	}
	if v, ok := os.LookupEnv("PORT_LS_STATE"); ok {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("PORT_LS_STATE must not be empty")
		}
		parts := strings.Split(v, ",")
		cfg.State = cfg.State[:0]
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				return fmt.Errorf("PORT_LS_STATE contains an empty value")
			}
			cfg.State = append(cfg.State, part)
		}
	}
	if v, ok := os.LookupEnv("PORT_LS_FREE_DEFAULT_START"); ok {
		p, err := strconv.Atoi(v)
		if err != nil {
			return fmt.Errorf("PORT_LS_FREE_DEFAULT_START: %w", err)
		}
		cfg.Free.DefaultStart = &p
	}
	if v, ok := os.LookupEnv("PORT_LS_FREE_DEFAULT_RANGE"); ok {
		cfg.Free.DefaultRange = &v
	}
	if cfg.Free.DefaultStart != nil && cfg.Free.DefaultRange != nil {
		return fmt.Errorf("PORT_LS_FREE_DEFAULT_START and PORT_LS_FREE_DEFAULT_RANGE are mutually exclusive")
	}
	return nil
}
