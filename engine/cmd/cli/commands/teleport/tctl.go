/*
2026 © Postgres.ai
*/

// Package teleport provides the teleport sidecar CLI command.
package teleport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

const tctlCommandTimeout = 30 * time.Second

// safeYAMLValue matches strings safe to embed in YAML: alphanumeric,
// hyphens, underscores, dots, colons, forward slashes, and at signs.
var safeYAMLValue = regexp.MustCompile(`^[a-zA-Z0-9._:/@\-]+$`)

// sanitizeYAMLValue validates that a string is safe to embed in a YAML value.
// It rejects values containing characters that could break YAML structure
// (newlines, quotes, braces, etc.) to prevent YAML injection.
func sanitizeYAMLValue(value, fieldName string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", fieldName)
	}

	if !safeYAMLValue.MatchString(value) {
		return "", fmt.Errorf(
			"%s contains invalid characters: only alphanumeric, hyphens, underscores, dots, colons, slashes are allowed",
			fieldName,
		)
	}

	return value, nil
}

// tctlDB represents a Teleport DB resource as returned by tctl get db --format=json.
type tctlDB struct {
	Metadata struct {
		Name   string            `json:"name"`
		Labels map[string]string `json:"labels"`
	} `json:"metadata"`
}

// dbResource holds parameters for creating a Teleport DB resource.
type dbResource struct {
	Name     string
	Port     int
	EnvID    string
	CloneID  string
	Username string
}

func runTctl(ctx context.Context, tctlPath, identityFile, proxyAddr string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, tctlCommandTimeout)
	defer cancel()

	baseArgs := []string{
		"--identity", identityFile,
		"--auth-server", proxyAddr,
	}
	fullArgs := append(baseArgs, args...)

	out, err := exec.CommandContext(ctx, tctlPath, fullArgs...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("tctl %s failed: %w: %s", strings.Join(args, " "), err, string(out))
	}

	return out, nil
}

func createDB(ctx context.Context, cfg *Config, res dbResource) error {
	yaml, err := buildDBYAML(res)
	if err != nil {
		return err
	}

	return runTctlCreate(ctx, cfg, yaml)
}

// buildDBYAML produces the Teleport DB resource YAML for a clone.
func buildDBYAML(res dbResource) ([]byte, error) {
	safeName, err := sanitizeYAMLValue(res.Name, "name")
	if err != nil {
		return nil, fmt.Errorf("invalid db resource: %w", err)
	}

	safeEnvID, err := sanitizeYAMLValue(res.EnvID, "environment")
	if err != nil {
		return nil, fmt.Errorf("invalid db resource: %w", err)
	}

	safeCloneID, err := sanitizeYAMLValue(res.CloneID, "clone_id")
	if err != nil {
		return nil, fmt.Errorf("invalid db resource: %w", err)
	}

	userLabel := ""

	if res.Username != "" {
		safeUsername, err := sanitizeYAMLValue(res.Username, "dblab_user")
		if err != nil {
			return nil, fmt.Errorf("invalid db resource: %w", err)
		}

		userLabel = fmt.Sprintf("\n    dblab_user: \"%s\"", safeUsername)
	}

	yaml := fmt.Sprintf(`kind: db
version: v3
metadata:
  name: "%s"
  labels:
    dblab: "true"
    environment: "%s"
    clone_id: "%s"%s
spec:
  protocol: postgres
  uri: "127.0.0.1:%d"
  tls:
    mode: insecure
`, safeName, safeEnvID, safeCloneID, userLabel, res.Port)

	return []byte(yaml), nil
}

func removeDB(ctx context.Context, cfg *Config, name string) error {
	if _, err := sanitizeYAMLValue(name, "name"); err != nil {
		return fmt.Errorf("invalid db resource: %w", err)
	}

	_, err := runTctl(ctx, cfg.TctlPath, cfg.TeleportIdentity, cfg.TeleportProxy, "rm", fmt.Sprintf("db/%s", name))

	return err
}

func createApp(ctx context.Context, cfg *Config, name, uri, envID string) error {
	safeName, err := sanitizeYAMLValue(name, "name")
	if err != nil {
		return fmt.Errorf("invalid app resource: %w", err)
	}

	safeURI, err := sanitizeYAMLValue(uri, "uri")
	if err != nil {
		return fmt.Errorf("invalid app resource: %w", err)
	}

	safeEnvID, err := sanitizeYAMLValue(envID, "environment")
	if err != nil {
		return fmt.Errorf("invalid app resource: %w", err)
	}

	yaml := fmt.Sprintf(`kind: app
version: v3
metadata:
  name: "%s"
  labels:
    dblab: "true"
    environment: "%s"
spec:
  uri: "%s"
  insecure_skip_verify: true # DBLab engine uses a self-signed certificate; connection is localhost-only
`, safeName, safeEnvID, safeURI)

	return runTctlCreate(ctx, cfg, []byte(yaml))
}

func listDBs(ctx context.Context, cfg *Config) (map[string]bool, error) {
	out, err := runTctl(ctx, cfg.TctlPath, cfg.TeleportIdentity, cfg.TeleportProxy, "get", "db", "--format=json")
	if err != nil {
		return nil, err
	}

	return parseListDBsOutput(out, cfg.EnvironmentID)
}

// parseListDBsOutput filters tctl JSON output for DBLab-managed databases
// matching the given environment.
func parseListDBsOutput(data []byte, envID string) (map[string]bool, error) {
	var dbs []tctlDB
	if err := json.Unmarshal(data, &dbs); err != nil {
		return nil, fmt.Errorf("failed to parse tctl output: %w", err)
	}

	result := make(map[string]bool, len(dbs))

	for _, db := range dbs {
		if db.Metadata.Labels["dblab"] == "true" && db.Metadata.Labels["environment"] == envID {
			result[db.Metadata.Name] = true
		}
	}

	return result, nil
}

func appExists(ctx context.Context, cfg *Config, name string) (bool, error) {
	_, err := runTctl(ctx, cfg.TctlPath, cfg.TeleportIdentity, cfg.TeleportProxy, "get", fmt.Sprintf("app/%s", name), "--format=json")
	if err == nil {
		return true, nil
	}

	if strings.Contains(err.Error(), "not found") {
		return false, nil
	}

	return false, err
}

func runTctlCreate(ctx context.Context, cfg *Config, yamlData []byte) error {
	baseArgs := []string{
		"--identity", cfg.TeleportIdentity,
		"--auth-server", cfg.TeleportProxy,
		"create", "-f", "-",
	}

	cmd := exec.CommandContext(ctx, cfg.TctlPath, baseArgs...)
	cmd.Stdin = bytes.NewReader(yamlData)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tctl create failed: %w: %s", err, string(out))
	}

	return nil
}
