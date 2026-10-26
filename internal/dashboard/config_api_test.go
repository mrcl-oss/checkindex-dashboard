package dashboard

import (
	"path/filepath"
	"testing"
)

func TestConfigAPILoadStarterWhenMissing(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "config.json")

	dto, err := LoadConfigDTO(path)
	if err != nil {
		t.Fatalf("LoadConfigDTO() error = %v", err)
	}
	if dto.DBName != "database" || dto.AuthSource != "admin" {
		t.Errorf("unexpected defaults: dbName=%s, authSource=%s", dto.DBName, dto.AuthSource)
	}
	if _, ok := dto.Cluster["default"]; !ok {
		t.Error("expected default cluster placeholder")
	}
}

func TestConfigAPISaveAndPreservePassword(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "config.json")

	initial := ConfigDTO{
		DBName:      "testdb",
		AuthSource:  "admin",
		HostMapping: map[string]string{"alias": "target"},
		Cluster: map[string]ClusterDTO{
			"primary": {
				InitialURI: "mongodb://127.0.0.1:27017",
				Username:   "appuser",
				Password:   "supersecret",
			},
		},
	}

	if err := SaveConfigDTO(path, initial); err != nil {
		t.Fatalf("SaveConfigDTO(initial) error = %v", err)
	}

	// Load it back
	loaded, err := LoadConfigDTO(path)
	if err != nil {
		t.Fatalf("LoadConfigDTO() error = %v", err)
	}
	if !loaded.Cluster["primary"].HasPassword {
		t.Error("expected HasPassword=true")
	}
	if loaded.Cluster["primary"].Password != "" {
		t.Error("expected Password to be blanked in DTO for security")
	}

	// Update DBName without resending password
	update := loaded
	update.DBName = "updated_db"
	if err := SaveConfigDTO(path, update); err != nil {
		t.Fatalf("SaveConfigDTO(update) error = %v", err)
	}

	// Verify on-disk config still has "supersecret"
	reloaded, err := LoadConfigDTO(path)
	if err != nil {
		t.Fatalf("LoadConfigDTO() after update error = %v", err)
	}
	if reloaded.DBName != "updated_db" {
		t.Errorf("DBName = %q, want updated_db", reloaded.DBName)
	}
	if !reloaded.Cluster["primary"].HasPassword {
		t.Error("password was lost during update")
	}
}

func TestConfigAPIValidationRejections(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "config.json")

	// Missing DBName
	err := SaveConfigDTO(path, ConfigDTO{
		DBName:     "",
		AuthSource: "admin",
		Cluster:    map[string]ClusterDTO{"c": {InitialURI: "mongodb://x", Username: "u"}},
	})
	if err == nil {
		t.Error("expected error for empty dbName")
	}

	// Empty clusters
	err = SaveConfigDTO(path, ConfigDTO{
		DBName:     "db",
		AuthSource: "admin",
		Cluster:    map[string]ClusterDTO{},
	})
	if err == nil {
		t.Error("expected error for empty cluster")
	}

	// Missing URI
	err = SaveConfigDTO(path, ConfigDTO{
		DBName:     "db",
		AuthSource: "admin",
		Cluster: map[string]ClusterDTO{
			"prod": {InitialURI: "", Username: "u"},
		},
	})
	if err == nil {
		t.Error("expected error for missing initialUri")
	}
}
