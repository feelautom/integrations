package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rclone/rclone/fs/config"
)

func CleanPlakarRcloneConf(configMap map[string]string) {
	delete(configMap, "location")
	for k, v := range configMap {
		if strings.HasPrefix(k, "rclone_") {
			newKey := strings.TrimPrefix(k, "rclone_")
			configMap[newKey] = v
			delete(configMap, k)
		}
	}
}

func WriteRcloneConfigFile(name string, remoteMap map[string]string) (*os.File, error) {
	file, err := createTempConf()
	if err != nil {
		return nil, err
	}

	baseConfigPath := extractBaseConfigPath(remoteMap)
	if err := copyExistingRcloneConfig(file, name, baseConfigPath); err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return nil, err
	}

	_, err = fmt.Fprintf(file, "[%s]\n", name)
	if err != nil {
		return nil, err
	}
	for k, v := range remoteMap {
		_, err = fmt.Fprintf(file, "%s = %s\n", k, v)
	}
	return file, nil
}

func extractBaseConfigPath(remoteMap map[string]string) string {
	for _, key := range []string{"config_file", "config"} {
		if path := remoteMap[key]; path != "" {
			delete(remoteMap, key)
			return path
		}
	}

	return ""
}

func copyExistingRcloneConfig(file *os.File, overrideSection string, explicitConfigPath string) error {
	configPath := explicitConfigPath
	if configPath == "" {
		configPath = findRcloneConfigPath()
	}
	if configPath == "" {
		return nil
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read rclone config file %s: %w", configPath, err)
	}

	withoutOverride := removeConfigSection(string(content), overrideSection)
	if strings.TrimSpace(withoutOverride) == "" {
		return nil
	}

	if _, err := fmt.Fprintln(file, strings.TrimRight(withoutOverride, "\r\n")); err != nil {
		return err
	}
	_, err = fmt.Fprintln(file)
	return err
}

func findRcloneConfigPath() string {
	candidates := []string{}

	if path := os.Getenv("RCLONE_CONFIG"); path != "" {
		candidates = append(candidates, path)
	}

	if configDir, err := os.UserConfigDir(); err == nil && configDir != "" {
		candidates = append(candidates, filepath.Join(configDir, "rclone", "rclone.conf"))
	}

	if homeDir, err := os.UserHomeDir(); err == nil && homeDir != "" {
		candidates = append(candidates, filepath.Join(homeDir, ".config", "rclone", "rclone.conf"))
	}

	for _, path := range candidates {
		if path == "" {
			continue
		}
		if stat, err := os.Stat(path); err == nil && !stat.IsDir() {
			return path
		}
	}

	return ""
}

func removeConfigSection(content, section string) string {
	var builder strings.Builder
	inSkippedSection := false

	for _, line := range strings.SplitAfter(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]") {
			currentSection := strings.TrimSuffix(strings.TrimPrefix(trimmed, "["), "]")
			inSkippedSection = currentSection == section
		}

		if !inSkippedSection {
			builder.WriteString(line)
		}
	}

	return builder.String()
}

func createTempConf() (*os.File, error) {
	tempFile, err := os.CreateTemp("", "rclone-*.conf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temporary config file: %w", err)
	}
	err = config.SetConfigPath(tempFile.Name())
	if err != nil {
		return nil, err
	}
	return tempFile, nil
}

func DeleteTempConf(name string) {
	err := os.Remove(name)
	if err != nil {
		fmt.Printf("Error removing temporary file: %v\n", err)
	}
}
