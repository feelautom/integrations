package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemoveConfigSection(t *testing.T) {
	input := "[gdrive]\n" +
		"type = drive\n" +
		"token = redacted\n\n" +
		"[crypt]\n" +
		"type = crypt\n" +
		"remote = old:bucket\n\n" +
		"[alias]\n" +
		"type = alias\n" +
		"remote = crypt:path\n"

	got := removeConfigSection(input, "crypt")

	if strings.Contains(got, "[crypt]") {
		t.Fatalf("expected crypt section to be removed, got:\n%s", got)
	}
	if !strings.Contains(got, "[gdrive]") || !strings.Contains(got, "[alias]") {
		t.Fatalf("expected unrelated sections to remain, got:\n%s", got)
	}
}

func TestWriteRcloneConfigFileCopiesExistingDependentSections(t *testing.T) {
	dir := t.TempDir()
	sourceConfig := filepath.Join(dir, "rclone.conf")
	t.Setenv("RCLONE_CONFIG", sourceConfig)

	err := os.WriteFile(sourceConfig, []byte(
		"[gdrive]\n"+
			"type = drive\n"+
			"token = redacted\n\n"+
			"[crypt]\n"+
			"type = crypt\n"+
			"remote = stale:path\n",
	), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	file, err := WriteRcloneConfigFile("crypt", map[string]string{
		"type":   "crypt",
		"remote": "gdrive:Workspace/Backups",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer DeleteTempConf(file.Name())
	defer file.Close()

	contentBytes, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, "[gdrive]") {
		t.Fatalf("expected dependent gdrive section to be copied, got:\n%s", content)
	}
	if strings.Contains(content, "stale:path") {
		t.Fatalf("expected imported crypt section to override existing crypt section, got:\n%s", content)
	}
	if !strings.Contains(content, "remote = gdrive:Workspace/Backups") {
		t.Fatalf("expected imported crypt remote to be written, got:\n%s", content)
	}
}

func TestWriteRcloneConfigFileSupportsExplicitConfigFile(t *testing.T) {
	dir := t.TempDir()
	sourceConfig := filepath.Join(dir, "custom-rclone.conf")
	t.Setenv("RCLONE_CONFIG", filepath.Join(dir, "missing-rclone.conf"))

	err := os.WriteFile(sourceConfig, []byte(
		"[gdrive]\n"+
			"type = drive\n"+
			"token = redacted\n",
	), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	file, err := WriteRcloneConfigFile("crypt", map[string]string{
		"config_file": sourceConfig,
		"type":        "crypt",
		"remote":      "gdrive:Workspace/Backups",
	})
	if err != nil {
		t.Fatal(err)
	}
	defer DeleteTempConf(file.Name())
	defer file.Close()

	contentBytes, err := os.ReadFile(file.Name())
	if err != nil {
		t.Fatal(err)
	}
	content := string(contentBytes)

	if !strings.Contains(content, "[gdrive]") {
		t.Fatalf("expected explicit config file sections to be copied, got:\n%s", content)
	}
	if strings.Contains(content, "config_file") {
		t.Fatalf("expected config_file to be consumed as a plugin option, got:\n%s", content)
	}
}
