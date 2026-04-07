package python

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Manager int

const (
	ManagerPip Manager = iota
	ManagerUV
)

func (m Manager) String() string {
	switch m {
	case ManagerUV:
		return "uv"
	default:
		return "pip"
	}
}

type Environment struct {
	PythonPath    string
	PythonVersion string
	Manager       Manager
}

func DetectEnvironment(pythonOverride, managerOverride string) (*Environment, error) {
	env := &Environment{}

	if err := env.detectPython(pythonOverride); err != nil {
		return nil, fmt.Errorf("detecting python: %w", err)
	}

	env.detectManager(managerOverride)

	return env, nil
}

func (e *Environment) detectPython(override string) error {
	if override != "" {
		e.PythonPath = override
		return e.fetchVersion()
	}

	// check for venv in current directory
	venvPython := filepath.Join(".venv", "bin", "python")
	if _, err := os.Stat(venvPython); err == nil {
		abs, err := filepath.Abs(venvPython)
		if err != nil {
			return fmt.Errorf("resolving venv path: %w", err)
		}
		e.PythonPath = abs
		return e.fetchVersion()
	}

	// fall back to PATH
	pythonPath, err := exec.LookPath("python3")
	if err != nil {
		pythonPath, err = exec.LookPath("python")
		if err != nil {
			return errors.New("python not found in PATH")
		}
	}

	e.PythonPath = pythonPath
	return e.fetchVersion()
}

func (e *Environment) fetchVersion() error {
	out, err := exec.CommandContext(context.Background(), e.PythonPath, "--version").Output() //nolint:gosec // path is user-provided or detected from system
	if err != nil {
		return fmt.Errorf("running %s --version: %w", e.PythonPath, err)
	}

	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) >= 2 {
		e.PythonVersion = parts[1]
	}

	return nil
}

func (e *Environment) detectManager(override string) {
	if override != "" {
		switch strings.ToLower(override) {
		case "uv":
			e.Manager = ManagerUV
		default:
			e.Manager = ManagerPip
		}
		return
	}

	if _, err := os.Stat("uv.lock"); err == nil {
		e.Manager = ManagerUV
		return
	}

	data, err := os.ReadFile("pyproject.toml")
	if err == nil && strings.Contains(string(data), "[tool.uv]") {
		e.Manager = ManagerUV
		return
	}

	e.Manager = ManagerPip
}

type Package struct {
	Name             string
	InstalledVersion string
	LatestVersion    string
}

func (e *Environment) ListPackages() ([]Package, error) {
	var cmd *exec.Cmd

	switch e.Manager {
	case ManagerUV:
		cmd = exec.CommandContext(context.Background(), "uv", "pip", "list", "--format=json", "--python", e.PythonPath) //nolint:gosec // path is user-provided or detected
	default:
		cmd = exec.CommandContext(context.Background(), e.PythonPath, "-m", "pip", "list", "--format=json") //nolint:gosec // path is user-provided or detected
	}

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("listing packages: %w", err)
	}

	var raw []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}

	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parsing package list: %w", err)
	}

	pkgs := make([]Package, 0, len(raw))
	for _, r := range raw {
		pkgs = append(pkgs, Package{
			Name:             r.Name,
			InstalledVersion: r.Version,
			LatestVersion:    "…",
		})
	}

	return pkgs, nil
}

func (e *Environment) InstallPackage(name, version string) error {
	target := name + "==" + version

	var cmd *exec.Cmd

	switch e.Manager {
	case ManagerUV:
		cmd = exec.CommandContext(context.Background(), "uv", "pip", "install", target, "--python", e.PythonPath) //nolint:gosec // user-initiated action
	default:
		cmd = exec.CommandContext(context.Background(), e.PythonPath, "-m", "pip", "install", target) //nolint:gosec // user-initiated action
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("installing %s: %w\n%s", target, err, output)
	}

	return nil
}
