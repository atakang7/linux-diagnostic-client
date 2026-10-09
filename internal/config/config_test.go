package config

import "testing"

func TestLoadHonorsEnvironment(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://test:test@db:5432/diagnostic")
	t.Setenv("SERVER_ADDR", "127.0.0.1:9080")
	t.Setenv("AGENT_ADDR", "127.0.0.1:9081")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgres://test:test@db:5432/diagnostic" {
		t.Fatalf("DATABASE_URL not applied")
	}
	if cfg.ServerAddr != "127.0.0.1:9080" || cfg.AgentAddr != "127.0.0.1:9081" {
		t.Fatalf("configured bind addresses not applied: %+v", cfg)
	}
}
