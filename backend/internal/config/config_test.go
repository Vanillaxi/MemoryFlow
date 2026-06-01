package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigReadsGithubConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := []byte("github:\n  token: secret\n  default_limit: 12\n  default_days: 5\n")
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Github.Token != "secret" || cfg.Github.DefaultLimit != 12 || cfg.Github.DefaultDays != 5 {
		t.Fatalf("unexpected github config: %#v", cfg.Github)
	}
}
