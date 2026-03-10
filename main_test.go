package main

import "testing"

func TestResolveConfigDir(t *testing.T) {
	t.Run("uses override when provided", func(t *testing.T) {
		got, err := resolveConfigDir("/tmp/custom-config")
		if err != nil {
			t.Fatalf("resolveConfigDir() error = %v", err)
		}
		if got != "/tmp/custom-config" {
			t.Fatalf("resolveConfigDir() = %q, want %q", got, "/tmp/custom-config")
		}
	})

	t.Run("uses user config dir when override is empty", func(t *testing.T) {
		got, err := resolveConfigDir("")
		if err != nil {
			t.Fatalf("resolveConfigDir() error = %v", err)
		}
		if got == "" {
			t.Fatal("resolveConfigDir() returned empty path")
		}
	})
}
