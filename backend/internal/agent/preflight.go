package agent

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/anthropics/acpone/internal/config"
)

// CheckResult holds the result of checking an agent
type CheckResult struct {
	AgentID string
	Status  string
	Error   error
}

// PreflightCheck checks all agents are available
func PreflightCheck(agents []config.AgentConfig) error {
	var wg sync.WaitGroup
	results := make(chan CheckResult, len(agents))

	for _, agent := range agents {
		wg.Add(1)
		go func(a config.AgentConfig) {
			defer wg.Done()
			result := checkAgent(a)
			results <- result
		}(agent)
	}

	wg.Wait()
	close(results)

	// Collect and print results
	var errs []string
	for result := range results {
		if result.Error != nil {
			fmt.Printf("   ✗ %s: %s\n", result.AgentID, result.Error)
			errs = append(errs, fmt.Sprintf("%s: %v", result.AgentID, result.Error))
		} else {
			fmt.Printf("   ✓ %s: %s\n", result.AgentID, result.Status)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("preflight failed:\n  %s", strings.Join(errs, "\n  "))
	}

	return nil
}

func checkAgent(agent config.AgentConfig) CheckResult {
	packageName := extractPackageName(agent)

	if packageName != "" {
		// NPX-based agent - check and install if needed
		status, err := ensurePackage(packageName)
		if err != nil {
			return CheckResult{AgentID: agent.ID, Error: err}
		}
		return CheckResult{AgentID: agent.ID, Status: status}
	}

	// Direct command - check if it exists
	err := commandExists(agent.Command)
	if err != nil {
		return CheckResult{AgentID: agent.ID, Error: err}
	}
	return CheckResult{AgentID: agent.ID, Status: fmt.Sprintf("%s found", agent.Command)}
}

func extractPackageName(agent config.AgentConfig) string {
	if agent.Command != "npx" || len(agent.Args) == 0 {
		return ""
	}

	// Skip flags like -y
	for _, arg := range agent.Args {
		if !strings.HasPrefix(arg, "-") {
			return arg
		}
	}
	return ""
}

func commandExists(command string) error {
	_, err := exec.LookPath(command)
	if err != nil {
		return fmt.Errorf("command not found: %s", command)
	}
	return nil
}

// ensurePackage checks if npm package is cached, installs if not
func ensurePackage(packageName string) (string, error) {
	// Check if already in npx cache
	if isPackageCached(packageName) {
		return fmt.Sprintf("%s (cached)", packageName), nil
	}

	// Not cached, need to install
	fmt.Printf("   ⏳ %s: installing...\n", packageName)

	err := installPackage(packageName)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s (installed)", packageName), nil
}

func isPackageCached(packageName string) bool {
	// Check if package exists in npm cache using npm cache ls
	// or check npx cache directory directly
	cmd := exec.Command("npm", "list", "-g", "--depth=0", packageName)
	err := cmd.Run()
	if err == nil {
		fmt.Printf("   🔍 %s: globally installed\n", packageName)
		return true
	}

	// Check npx cache by looking for the package in ~/.npm/_npx
	home, _ := os.UserHomeDir()
	npxCacheDir := filepath.Join(home, ".npm", "_npx")

	// Walk through npx cache directories to find package
	if entries, err := os.ReadDir(npxCacheDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				pkgJsonPath := filepath.Join(npxCacheDir, entry.Name(), "node_modules", packageName, "package.json")
				if _, err := os.Stat(pkgJsonPath); err == nil {
					fmt.Printf("   🔍 %s: found in npx cache\n", packageName)
					return true
				}
			}
		}
	}

	fmt.Printf("   🔍 %s: not cached\n", packageName)
	return false
}

func installPackage(packageName string) error {
	// Use npx -y to auto-install
	cmdStr := fmt.Sprintf("npx -y %s --help", packageName)
	fmt.Printf("   📦 Installing: %s\n", cmdStr)

	cmd := exec.Command("npx", "-y", packageName, "--help")
	output, err := cmd.CombinedOutput()

	outputStr := string(output)

	// Check for npm errors
	if strings.Contains(outputStr, "npm ERR!") || strings.Contains(outputStr, "404 Not Found") {
		return fmt.Errorf("failed to install: %s", strings.TrimSpace(outputStr))
	}

	// npx downloads regardless of exit code, so we only fail on npm errors
	if err != nil && strings.Contains(outputStr, "npm ERR!") {
		return fmt.Errorf("failed to install: %w", err)
	}

	return nil
}
