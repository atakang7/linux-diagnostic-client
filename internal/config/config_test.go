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

func TestLoadRejectsInvalidEnvironment(t *testing.T) {
	cases := []struct {
		key   string
		value string
	}{
		{"DATABASE_URL", ""},
		{"SERVER_ADDR", ""},
		{"AGENT_ADDR", "no-port"},
		{"AGENT_ADDR", "127.0.0.1:99999"},
		{"SERVER_ADDR", "127.0.0.1:0"},
	}
	for _, tc := range cases {
		t.Run(tc.key+"="+tc.value, func(t *testing.T) {
			t.Setenv("DATABASE_URL", "postgres://example")
			t.Setenv("SERVER_ADDR", "127.0.0.1:8080")
			t.Setenv("AGENT_ADDR", "127.0.0.1:8081")
			t.Setenv(tc.key, tc.value)
			if _, err := Load(); err == nil {
				t.Fatalf("expected error for %s=%q", tc.key, tc.value)
			}
		})
	}
}
