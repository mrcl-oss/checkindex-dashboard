package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestLoadUsesSafeDefaults(t *testing.T) {
	for _, values := range []map[string]string{
		{},
		{ListenAddressEnv: " ", RefreshIntervalEnv: "\t"},
	} {
		got, err := load(lookup(values))
		if err != nil {
			t.Fatalf("load() error = %v", err)
		}
		if got.ListenAddress != DefaultListenAddress {
			t.Errorf("ListenAddress = %q, want %q", got.ListenAddress, DefaultListenAddress)
		}
		if got.RefreshInterval != time.Minute {
			t.Errorf("RefreshInterval = %v, want %v", got.RefreshInterval, time.Minute)
		}
	}
}

func TestLoadAcceptsValidOverrides(t *testing.T) {
	got, err := load(lookup(map[string]string{
		ListenAddressEnv:   "[::1]:9090",
		RefreshIntervalEnv: "2m30s",
	}))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if got.ListenAddress != "[::1]:9090" {
		t.Errorf("ListenAddress = %q, want %q", got.ListenAddress, "[::1]:9090")
	}
	if got.RefreshInterval != 150*time.Second {
		t.Errorf("RefreshInterval = %v, want %v", got.RefreshInterval, 150*time.Second)
	}
}

func TestLoadReadsEnvironmentConfiguration(t *testing.T) {
	t.Setenv(ListenAddressEnv, "127.0.0.1:8090")
	t.Setenv(RefreshIntervalEnv, "45s")

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got.ListenAddress != "127.0.0.1:8090" || got.RefreshInterval != 45*time.Second {
		t.Fatalf("Load() = %+v, want listener 127.0.0.1:8090 and refresh interval 45s", got)
	}
}

func TestLoadRejectsInvalidOverridesWithoutEchoingValues(t *testing.T) {
	tests := []struct {
		name   string
		values map[string]string
		secret string
	}{
		{
			name:   "malformed listener address",
			values: map[string]string{ListenAddressEnv: "not-an-address"},
			secret: "not-an-address",
		},
		{
			name:   "listener port out of range",
			values: map[string]string{ListenAddressEnv: "127.0.0.1:70000"},
			secret: "70000",
		},
		{
			name:   "unparseable refresh interval",
			values: map[string]string{RefreshIntervalEnv: "private-value"},
			secret: "private-value",
		},
		{
			name:   "zero refresh interval",
			values: map[string]string{RefreshIntervalEnv: "0s"},
		},
		{
			name:   "negative refresh interval",
			values: map[string]string{RefreshIntervalEnv: "-1s"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := load(lookup(tt.values))
			if err == nil {
				t.Fatal("load() error = nil, want invalid configuration error")
			}
			if tt.secret != "" && strings.Contains(err.Error(), tt.secret) {
				t.Fatalf("configuration error exposed the supplied value: %v", err)
			}
		})
	}
}

func TestLoadAcceptsPortEnvOverride(t *testing.T) {
	got, err := load(lookup(map[string]string{
		PortEnv: "9090",
	}))
	if err != nil {
		t.Fatalf("load() error = %v", err)
	}
	if got.ListenAddress != "127.0.0.1:9090" {
		t.Errorf("ListenAddress = %q, want %q", got.ListenAddress, "127.0.0.1:9090")
	}
}

func TestLoadDotEnvParsesVariables(t *testing.T) {
	tempDir := t.TempDir()
	envPath := tempDir + "/.env"
	content := "# comment line\nPORT=7777\nCHECKINDEX_DASHBOARD_REFRESH_INTERVAL=\"45s\"\n"
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	loadDotEnv(envPath)

	if val := os.Getenv("PORT"); val != "7777" {
		t.Errorf("PORT = %q, want 7777", val)
	}
	if val := os.Getenv("CHECKINDEX_DASHBOARD_REFRESH_INTERVAL"); val != "45s" {
		t.Errorf("CHECKINDEX_DASHBOARD_REFRESH_INTERVAL = %q, want 45s", val)
	}
}

func lookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
