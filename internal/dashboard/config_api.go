package dashboard

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mrcl-oss/checkIndex/config"
)

// ConfigDTO represents the editable configuration for the UI.
type ConfigDTO struct {
	DBName      string                `json:"dbName"`
	AuthSource  string                `json:"authSource"`
	HostMapping map[string]string     `json:"hostMapping"`
	Cluster     map[string]ClusterDTO `json:"cluster"`
}

// ClusterDTO represents an individual cluster profile's editable fields.
type ClusterDTO struct {
	InitialURI  string            `json:"initialUri"`
	Username    string            `json:"username"`
	Password    string            `json:"password,omitempty"`
	HasPassword bool              `json:"hasPassword"`
	HostMapping map[string]string `json:"hostMapping"`
}

// DefaultConfigPath returns the canonical path to ~/.checkIndex/config.json,
// honoring the CHECKINDEX_CONFIG_PATH override if set.
func DefaultConfigPath() (string, error) {
	if custom := strings.TrimSpace(os.Getenv("CHECKINDEX_CONFIG_PATH")); custom != "" {
		return custom, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home directory: %w", err)
	}
	return filepath.Join(home, ".checkIndex", "config.json"), nil
}

// LoadConfigDTO loads and adapts configuration for the UI.
// If the configuration file does not exist yet, a default template is returned.
func LoadConfigDTO(path string) (ConfigDTO, error) {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return ConfigDTO{}, err
		}
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return defaultStarterConfig(), nil
	}

	loaded, err := config.LoadConfig(path)
	if err != nil {
		// If loadConfig just created the file because it was missing, retry loading it
		if strings.Contains(err.Error(), "created ") {
			loaded, err = config.LoadConfig(path)
		}
	}
	if err != nil {
		return ConfigDTO{}, fmt.Errorf("load config: %w", err)
	}

	dto := ConfigDTO{
		DBName:      loaded.DBName,
		AuthSource:  loaded.AuthSource,
		HostMapping: make(map[string]string),
		Cluster:     make(map[string]ClusterDTO),
	}
	for k, v := range loaded.HostMapping {
		dto.HostMapping[k] = v
	}

	for name, creds := range loaded.Cluster {
		hm := make(map[string]string)
		for k, v := range creds.HostMapping {
			hm[k] = v
		}
		dto.Cluster[name] = ClusterDTO{
			InitialURI:  creds.InitialURI,
			Username:    creds.Username,
			HasPassword: strings.TrimSpace(creds.Password) != "",
			HostMapping: hm,
		}
	}

	if len(dto.Cluster) == 0 {
		dto.Cluster["default"] = ClusterDTO{
			HostMapping: make(map[string]string),
		}
	}

	return dto, nil
}

// SaveConfigDTO validates and saves configuration to disk.
// If an existing profile's password is submitted empty, the existing password is preserved.
func SaveConfigDTO(path string, dto ConfigDTO) error {
	if path == "" {
		var err error
		path, err = DefaultConfigPath()
		if err != nil {
			return err
		}
	}

	dbName := strings.TrimSpace(dto.DBName)
	authSource := strings.TrimSpace(dto.AuthSource)
	if dbName == "" {
		return errors.New("dbName is required")
	}
	if authSource == "" {
		return errors.New("authSource is required")
	}
	if len(dto.Cluster) == 0 {
		return errors.New("at least one cluster profile is required")
	}

	var existing config.Config
	if loaded, err := config.LoadConfig(path); err == nil {
		existing = loaded
	}

	target := config.Config{
		DBName:      dbName,
		AuthSource:  authSource,
		HostMapping: make(map[string]string),
		Cluster:     make(map[string]config.Credentials),
	}

	for k, v := range dto.HostMapping {
		tk, tv := strings.TrimSpace(k), strings.TrimSpace(v)
		if tk != "" && tv != "" {
			target.HostMapping[tk] = tv
		}
	}

	for name, credsDTO := range dto.Cluster {
		profName := strings.TrimSpace(name)
		if profName == "" {
			return errors.New("cluster profile name cannot be empty")
		}
		initialURI := strings.TrimSpace(credsDTO.InitialURI)
		if initialURI == "" {
			return fmt.Errorf("cluster profile %q requires initialUri", profName)
		}
		username := strings.TrimSpace(credsDTO.Username)
		if username == "" {
			return fmt.Errorf("cluster profile %q requires username", profName)
		}

		password := credsDTO.Password
		if password == "" {
			if existingProfile, exists := existing.Cluster[profName]; exists {
				password = existingProfile.Password
			}
		}

		hm := make(map[string]string)
		for k, v := range credsDTO.HostMapping {
			tk, tv := strings.TrimSpace(k), strings.TrimSpace(v)
			if tk != "" && tv != "" {
				hm[tk] = tv
			}
		}

		target.Cluster[profName] = config.Credentials{
			InitialURI:  initialURI,
			Username:    username,
			Password:    password,
			HostMapping: hm,
		}
	}

	if err := config.ValidateConfig(target); err != nil {
		return fmt.Errorf("validate config: %w", err)
	}

	if err := config.SaveConfig(path, target); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	return nil
}

func defaultStarterConfig() ConfigDTO {
	return ConfigDTO{
		DBName:      "database",
		AuthSource:  "admin",
		HostMapping: make(map[string]string),
		Cluster: map[string]ClusterDTO{
			"default": {
				InitialURI:  "",
				Username:    "",
				HasPassword: false,
				HostMapping: make(map[string]string),
			},
		},
	}
}

