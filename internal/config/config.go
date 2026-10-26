package config

import (
	"errors"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/mrcl-oss/checkindex-dashboard/internal/dashboard"
)

const (
	ListenAddressEnv     = "CHECKINDEX_DASHBOARD_LISTEN_ADDR"
	PortEnv              = "PORT"
	RefreshIntervalEnv   = "CHECKINDEX_DASHBOARD_REFRESH_INTERVAL"
	DefaultListenAddress = "127.0.0.1:6767"
)

// Config contains the dashboard's non-secret runtime settings.
type Config struct {
	ListenAddress   string
	RefreshInterval time.Duration
}

// Load reads and validates dashboard settings from .env (if present) and the process environment.
func Load() (Config, error) {
	loadDotEnv(".env")
	return load(os.LookupEnv)
}

func load(lookup func(string) (string, bool)) (Config, error) {
	cfg := Config{
		ListenAddress:   DefaultListenAddress,
		RefreshInterval: dashboard.DefaultRefreshInterval,
	}

	if value, ok := lookup(ListenAddressEnv); ok && strings.TrimSpace(value) != "" {
		cfg.ListenAddress = strings.TrimSpace(value)
	} else if portVal, ok := lookup(PortEnv); ok && strings.TrimSpace(portVal) != "" {
		p := strings.TrimSpace(portVal)
		if strings.Contains(p, ":") {
			cfg.ListenAddress = p
		} else {
			host, _, err := net.SplitHostPort(DefaultListenAddress)
			if err != nil {
				host = "127.0.0.1"
			}
			cfg.ListenAddress = net.JoinHostPort(host, p)
		}
	}
	if err := validateListenAddress(cfg.ListenAddress); err != nil {
		return Config{}, err
	}

	if value, ok := lookup(RefreshIntervalEnv); ok && strings.TrimSpace(value) != "" {
		interval, err := time.ParseDuration(strings.TrimSpace(value))
		if err != nil || interval <= 0 {
			return Config{}, errors.New(RefreshIntervalEnv + " must be a positive duration, such as 60s")
		}
		cfg.RefreshInterval = interval
	}

	return cfg, nil
}

func validateListenAddress(address string) error {
	_, portText, err := net.SplitHostPort(address)
	if err != nil {
		return errors.New(ListenAddressEnv + " must be a host:port address")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return errors.New(ListenAddressEnv + " port must be between 1 and 65535")
	}
	return nil
}

func loadDotEnv(filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if _, exists := os.LookupEnv(key); !exists && key != "" {
			_ = os.Setenv(key, val)
		}
	}
}
