package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "rosactl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override the home directory for testing
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Test saving a config
	testURL := "https://api.example.com"
	err = SetPlatformAPIURL(testURL)
	if err != nil {
		t.Fatalf("Failed to set platform API URL: %v", err)
	}

	// Verify the config file was created
	configPath := filepath.Join(tmpDir, configDir, configFile)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatalf("Config file was not created at %s", configPath)
	}

	// Test loading the config
	url, err := GetPlatformAPIURL()
	if err != nil {
		t.Fatalf("Failed to get platform API URL: %v", err)
	}

	if url != testURL {
		t.Errorf("Expected URL %q, got %q", testURL, url)
	}
}

func TestGetPlatformAPIURL_NotConfigured(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "rosactl-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Override the home directory for testing
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", tmpDir)

	// Test getting URL when not configured
	_, err = GetPlatformAPIURL()
	if err == nil {
		t.Error("Expected error when platform API URL is not configured")
	}
}
