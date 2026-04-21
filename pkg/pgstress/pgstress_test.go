package pgstress

import (
	"testing"
	"time"
)

func TestOptionsValidate(t *testing.T) {
	opts := Options{
		Host:           "localhost",
		Port:           "5432",
		User:           "postgres",
		Password:       "secret",
		Database:       "postgres",
		Connections:    5,
		KeepAlive:      10 * time.Second,
		RestartCheck:   30 * time.Second,
		ConnectTimeout: 5 * time.Second,
	}

	if err := opts.Validate(); err != nil {
		t.Fatalf("expected valid options, got %v", err)
	}
}

func TestBuildDSN(t *testing.T) {
	dsn, err := buildDSN("localhost", "5432", "postgres", "secret", "postgres", "require")
	if err != nil {
		t.Fatalf("unexpected error building DSN: %v", err)
	}
	if dsn == "" {
		t.Fatal("expected non-empty DSN")
	}
}
