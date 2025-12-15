package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/anthropics/acpone/internal/sysutil"
)

// DependencyItem represents a single dependency check item
type DependencyItem struct {
	Name    string `json:"name"`
	Command string `json:"command,omitempty"`
	Package string `json:"package,omitempty"`
	Status  string `json:"status"` // "checking", "ready", "missing", "not_installed", "installing", "error", "blocked"
	Message string `json:"message,omitempty"`
	Install string `json:"install,omitempty"`
}

// SetupStatus represents the overall setup status
type SetupStatus struct {
	Ready       bool             `json:"ready"`
	Environment []DependencyItem `json:"environment"` // npm, npx
	Agents      []DependencyItem `json:"agents"`      // claude, codex commands
	ACPPackages []DependencyItem `json:"acpPackages"` // @zed-industries/xxx-acp
}

// Install instructions for common tools
var installInstructions = map[string]string{
	"npm":    "Install Node.js from https://nodejs.org/",
	"npx":    "Install Node.js from https://nodejs.org/",
	"claude": "npm install -g @anthropic-ai/claude-code",
	"codex":  "npm install -g @openai/codex",
}

// Agent command to npm package mapping (for auto-install)
var agentNpmPackages = map[string]string{
	"claude": "@anthropic-ai/claude-code",
	"codex":  "@openai/codex",
}

// ACP package to agent command mapping
var acpToAgentCommand = map[string]struct {
	Name    string
	Command string
}{
	"@zed-industries/claude-code-acp": {Name: "Claude", Command: "claude"},
	"@zed-industries/codex-acp":       {Name: "Codex", Command: "codex"},
}

// initSetupStatus initializes status with all checks in "checking" state
func (s *Server) initSetupStatus() {
	// Environment: npm, npx
	env := []DependencyItem{
		{Name: "npm", Command: "npm", Status: "checking", Message: "Checking..."},
		{Name: "npx", Command: "npx", Status: "checking", Message: "Checking..."},
	}

	// Collect ACP packages and their required agent commands
	acpPkgs := []DependencyItem{}
	requiredAgents := map[string]struct {
		Name    string
		Command string
	}{}

	for _, agent := range s.config.Agents {
		if agent.Command == "npx" {
			pkgName := extractPackageName(agent.Command, agent.Args)
			if pkgName != "" {
				acpPkgs = append(acpPkgs, DependencyItem{
					Name:    agent.Name,
					Package: pkgName,
					Status:  "checking",
					Message: "Waiting...",
				})
				// Check if this ACP package requires an agent command
				if agentInfo, ok := acpToAgentCommand[pkgName]; ok {
					requiredAgents[agentInfo.Command] = agentInfo
				}
			}
		} else {
			// Non-npx command, add to required agents
			requiredAgents[agent.Command] = struct {
				Name    string
				Command string
			}{Name: agent.Command, Command: agent.Command}
		}
	}

	// Build agents list from required agents
	agents := []DependencyItem{}
	for _, agentInfo := range requiredAgents {
		agents = append(agents, DependencyItem{
			Name:    agentInfo.Name,
			Command: agentInfo.Command,
			Status:  "checking",
			Message: "Checking...",
		})
	}

	s.setupMu.Lock()
	s.setupStatus = &SetupStatus{
		Ready:       false,
		Environment: env,
		Agents:      agents,
		ACPPackages: acpPkgs,
	}
	s.setupMu.Unlock()
}

// checkDependenciesAsync checks all dependencies asynchronously
func (s *Server) checkDependenciesAsync() {
	npmReady := false
	npxReady := false

	// Phase 1: Check environment (npm, npx)
	for i := 0; i < 2; i++ {
		s.setupMu.RLock()
		item := s.setupStatus.Environment[i]
		s.setupMu.RUnlock()

		exists := commandExists(item.Command)

		s.setupMu.Lock()
		if exists {
			s.setupStatus.Environment[i].Status = "ready"
			s.setupStatus.Environment[i].Message = "Installed"
			if item.Command == "npm" {
				npmReady = true
			}
			if item.Command == "npx" {
				npxReady = true
			}
		} else {
			s.setupStatus.Environment[i].Status = "missing"
			s.setupStatus.Environment[i].Message = "Not found"
			if inst, ok := installInstructions[item.Command]; ok {
				s.setupStatus.Environment[i].Install = inst
			}
		}
		s.setupMu.Unlock()
		s.broadcastSetupStatus()
	}

	// Phase 2: Check agent commands (claude, codex)
	s.setupMu.RLock()
	agentCount := len(s.setupStatus.Agents)
	s.setupMu.RUnlock()

	allAgentsReady := true
	for i := 0; i < agentCount; i++ {
		s.setupMu.RLock()
		item := s.setupStatus.Agents[i]
		s.setupMu.RUnlock()

		exists := commandExists(item.Command)

		s.setupMu.Lock()
		if exists {
			s.setupStatus.Agents[i].Status = "ready"
			s.setupStatus.Agents[i].Message = "Installed"
		} else {
			s.setupStatus.Agents[i].Status = "missing"
			s.setupStatus.Agents[i].Message = "Not found"
			if inst, ok := installInstructions[item.Command]; ok {
				s.setupStatus.Agents[i].Install = inst
			}
			allAgentsReady = false
		}
		s.setupMu.Unlock()
		s.broadcastSetupStatus()
	}

	// Phase 3: Check ACP packages
	s.setupMu.RLock()
	acpCount := len(s.setupStatus.ACPPackages)
	s.setupMu.RUnlock()

	allACPReady := true
	for i := 0; i < acpCount; i++ {
		s.setupMu.RLock()
		item := s.setupStatus.ACPPackages[i]
		s.setupMu.RUnlock()

		var status, message string
		if !npmReady || !npxReady {
			status = "blocked"
			message = "Requires npm/npx"
			allACPReady = false
		} else if isPackageCached(item.Package) {
			status = "ready"
			message = "Cached"
		} else {
			status = "not_installed"
			message = "Not installed"
			allACPReady = false
		}

		s.setupMu.Lock()
		s.setupStatus.ACPPackages[i].Status = status
		s.setupStatus.ACPPackages[i].Message = message
		s.setupMu.Unlock()
		s.broadcastSetupStatus()
	}

	// Final ready state
	envReady := npmReady && npxReady
	s.setupMu.Lock()
	s.setupStatus.Ready = envReady && allAgentsReady && allACPReady
	s.setupMu.Unlock()
	s.broadcastSetupStatus()
}

// broadcastSetupStatus sends current status to all subscribers
func (s *Server) broadcastSetupStatus() {
	s.setupMu.RLock()
	status := SetupStatus{
		Ready:       s.setupStatus.Ready,
		Environment: append([]DependencyItem{}, s.setupStatus.Environment...),
		Agents:      append([]DependencyItem{}, s.setupStatus.Agents...),
		ACPPackages: append([]DependencyItem{}, s.setupStatus.ACPPackages...),
	}
	s.setupMu.RUnlock()

	s.setupSubsMu.RLock()
	for ch := range s.setupSubs {
		select {
		case ch <- status:
		default:
		}
	}
	s.setupSubsMu.RUnlock()
}

func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.setupMu.RLock()
	status := s.setupStatus
	s.setupMu.RUnlock()

	writeJSON(w, status)
}

func (s *Server) handleSetupSubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	ch := make(chan SetupStatus, 10)
	s.setupSubsMu.Lock()
	s.setupSubs[ch] = struct{}{}
	s.setupSubsMu.Unlock()

	defer func() {
		s.setupSubsMu.Lock()
		delete(s.setupSubs, ch)
		s.setupSubsMu.Unlock()
		close(ch)
	}()

	// Re-initialize and re-check dependencies on each subscribe
	s.initSetupStatus()
	go s.checkDependenciesAsync()

	// Send current status (checking state)
	s.setupMu.RLock()
	currentStatus := SetupStatus{
		Ready:       s.setupStatus.Ready,
		Environment: append([]DependencyItem{}, s.setupStatus.Environment...),
		Agents:      append([]DependencyItem{}, s.setupStatus.Agents...),
		ACPPackages: append([]DependencyItem{}, s.setupStatus.ACPPackages...),
	}
	s.setupMu.RUnlock()

	jsonData, _ := json.Marshal(currentStatus)
	fmt.Fprintf(w, "data: %s\n\n", jsonData)
	flusher.Flush()

	for {
		select {
		case status := <-ch:
			jsonData, _ := json.Marshal(status)
			fmt.Fprintf(w, "data: %s\n\n", jsonData)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (s *Server) handleSetupInstall(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return
	}

	sendEvent := func(eventType string, data any) {
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, jsonData)
		flusher.Flush()
	}

	// Check environment first
	if !commandExists("npm") || !commandExists("npx") {
		sendEvent("done", map[string]any{
			"success": false,
			"error":   "npm and npx are required. Please install Node.js first.",
		})
		return
	}

	allSuccess := true

	// Phase 1: Install missing agent commands (claude, codex)
	s.setupMu.RLock()
	agentCount := len(s.setupStatus.Agents)
	s.setupMu.RUnlock()

	for i := 0; i < agentCount; i++ {
		s.setupMu.RLock()
		item := s.setupStatus.Agents[i]
		s.setupMu.RUnlock()

		if item.Status == "ready" {
			sendEvent("progress", map[string]any{
				"index":   i,
				"type":    "agent",
				"status":  "ready",
				"message": "Already installed",
			})
			continue
		}

		// Check if we can install this agent
		npmPkg, canInstall := agentNpmPackages[item.Command]
		if !canInstall {
			sendEvent("progress", map[string]any{
				"index":   i,
				"type":    "agent",
				"status":  "error",
				"message": "Cannot auto-install",
			})
			allSuccess = false
			continue
		}

		// Update to installing
		s.setupMu.Lock()
		s.setupStatus.Agents[i].Status = "installing"
		s.setupStatus.Agents[i].Message = "Installing..."
		s.setupMu.Unlock()
		s.broadcastSetupStatus()

		sendEvent("progress", map[string]any{
			"index":   i,
			"type":    "agent",
			"status":  "installing",
			"message": fmt.Sprintf("Installing %s...", npmPkg),
		})

		err := installGlobalPackage(npmPkg, func(msg string) {
			sendEvent("log", map[string]any{
				"index":   i,
				"type":    "agent",
				"message": msg,
			})
		})

		if err != nil {
			s.setupMu.Lock()
			s.setupStatus.Agents[i].Status = "error"
			s.setupStatus.Agents[i].Message = err.Error()
			s.setupMu.Unlock()
			s.broadcastSetupStatus()

			sendEvent("progress", map[string]any{
				"index":   i,
				"type":    "agent",
				"status":  "error",
				"message": err.Error(),
			})
			allSuccess = false
		} else {
			s.setupMu.Lock()
			s.setupStatus.Agents[i].Status = "ready"
			s.setupStatus.Agents[i].Message = "Installed"
			s.setupMu.Unlock()
			s.broadcastSetupStatus()

			sendEvent("progress", map[string]any{
				"index":   i,
				"type":    "agent",
				"status":  "ready",
				"message": "Installed",
			})
		}
	}

	// Phase 2: Install ACP packages
	s.setupMu.RLock()
	acpCount := len(s.setupStatus.ACPPackages)
	s.setupMu.RUnlock()

	for i := 0; i < acpCount; i++ {
		s.setupMu.RLock()
		item := s.setupStatus.ACPPackages[i]
		s.setupMu.RUnlock()

		if item.Status == "ready" {
			sendEvent("progress", map[string]any{
				"index":   i,
				"type":    "acp",
				"status":  "ready",
				"message": "Already installed",
			})
			continue
		}

		// Update to installing
		s.setupMu.Lock()
		s.setupStatus.ACPPackages[i].Status = "installing"
		s.setupStatus.ACPPackages[i].Message = "Installing..."
		s.setupMu.Unlock()
		s.broadcastSetupStatus()

		sendEvent("progress", map[string]any{
			"index":   i,
			"type":    "acp",
			"status":  "installing",
			"message": fmt.Sprintf("Installing %s...", item.Package),
		})

		err := installPackageWithProgress(item.Package, func(msg string) {
			sendEvent("log", map[string]any{
				"index":   i,
				"type":    "acp",
				"message": msg,
			})
		})

		if err != nil {
			s.setupMu.Lock()
			s.setupStatus.ACPPackages[i].Status = "error"
			s.setupStatus.ACPPackages[i].Message = err.Error()
			s.setupMu.Unlock()
			s.broadcastSetupStatus()

			sendEvent("progress", map[string]any{
				"index":   i,
				"type":    "acp",
				"status":  "error",
				"message": err.Error(),
			})
			allSuccess = false
		} else {
			s.setupMu.Lock()
			s.setupStatus.ACPPackages[i].Status = "ready"
			s.setupStatus.ACPPackages[i].Message = "Installed"
			s.setupMu.Unlock()
			s.broadcastSetupStatus()

			sendEvent("progress", map[string]any{
				"index":   i,
				"type":    "acp",
				"status":  "ready",
				"message": "Installed",
			})
		}
	}

	// Update final ready state
	s.setupMu.Lock()
	s.setupStatus.Ready = allSuccess
	s.setupMu.Unlock()
	s.broadcastSetupStatus()

	sendEvent("done", map[string]any{"success": allSuccess})
}

func extractPackageName(command string, args []string) string {
	if command != "npx" || len(args) == 0 {
		return ""
	}
	for _, arg := range args {
		if !strings.HasPrefix(arg, "-") {
			return arg
		}
	}
	return ""
}

func isPackageCached(packageName string) bool {
	cmd := exec.Command("npm", "list", "-g", "--depth=0", packageName)
	sysutil.HideWindow(cmd)
	if err := cmd.Run(); err == nil {
		return true
	}

	home, _ := os.UserHomeDir()
	npxCacheDir := filepath.Join(home, ".npm", "_npx")

	entries, err := os.ReadDir(npxCacheDir)
	if err != nil {
		return false
	}

	for _, entry := range entries {
		if entry.IsDir() {
			pkgPath := filepath.Join(npxCacheDir, entry.Name(), "node_modules", packageName, "package.json")
			if _, err := os.Stat(pkgPath); err == nil {
				return true
			}
		}
	}
	return false
}

func commandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// npm registry URLs
var npmRegistries = []struct {
	Name string
	URL  string
}{
	{"China (npmmirror)", "https://registry.npmmirror.com"},
	{"Official (npmjs)", "https://registry.npmjs.org"},
}

// cachedRegistry stores the fastest registry URL
var (
	cachedRegistry   string
	registryOnce     sync.Once
	registryTestOnce sync.Once
)

// selectFastestRegistry tests registries and returns the fastest one
func selectFastestRegistry() string {
	registryOnce.Do(func() {
		log.Println("[Setup] Testing npm registry speeds...")

		type result struct {
			url      string
			name     string
			duration time.Duration
			err      error
		}

		results := make(chan result, len(npmRegistries))

		for _, reg := range npmRegistries {
			go func(name, url string) {
				start := time.Now()
				client := &http.Client{Timeout: 5 * time.Second}
				resp, err := client.Get(url + "/-/ping")
				duration := time.Since(start)

				if err != nil {
					results <- result{url: url, name: name, duration: duration, err: err}
					return
				}
				resp.Body.Close()

				results <- result{url: url, name: name, duration: duration, err: nil}
			}(reg.Name, reg.URL)
		}

		var fastest result
		fastest.duration = time.Hour // Start with a very long duration

		for i := 0; i < len(npmRegistries); i++ {
			r := <-results
			if r.err != nil {
				log.Printf("[Setup]   %s: failed (%v)", r.name, r.err)
				continue
			}
			log.Printf("[Setup]   %s: %v", r.name, r.duration.Round(time.Millisecond))
			if r.duration < fastest.duration {
				fastest = r
			}
		}

		if fastest.url != "" {
			cachedRegistry = fastest.url
			log.Printf("[Setup] Selected registry: %s (%s)", fastest.name, fastest.url)
		} else {
			cachedRegistry = npmRegistries[1].URL // Fallback to official
			log.Printf("[Setup] All registries failed, using official: %s", cachedRegistry)
		}
	})

	return cachedRegistry
}

func installPackageWithProgress(packageName string, logFn func(string)) error {
	registry := selectFastestRegistry()

	cmdStr := fmt.Sprintf("npx -y --registry=%s %s --help", registry, packageName)
	log.Printf("[Setup] Installing ACP package: %s", packageName)
	log.Printf("[Setup] Command: %s", cmdStr)
	logFn(fmt.Sprintf("Running: %s", cmdStr))

	cmd := exec.Command("npx", "-y", "--registry="+registry, packageName, "--help")
	sysutil.HideWindow(cmd)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if outputStr != "" {
		for _, line := range strings.Split(outputStr, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				log.Printf("[Setup]   %s", line)
			}
		}
	}

	if strings.Contains(outputStr, "npm ERR!") || strings.Contains(outputStr, "404 Not Found") {
		log.Printf("[Setup] Failed to install %s", packageName)
		return fmt.Errorf("failed to install: %s", strings.TrimSpace(outputStr))
	}

	if err != nil && strings.Contains(outputStr, "npm ERR!") {
		log.Printf("[Setup] Failed to install %s: %v", packageName, err)
		return fmt.Errorf("install failed: %w", err)
	}

	log.Printf("[Setup] Successfully installed %s", packageName)
	logFn("Installation completed")
	return nil
}

func installGlobalPackage(packageName string, logFn func(string)) error {
	registry := selectFastestRegistry()

	// First, try to uninstall existing package to avoid ENOTEMPTY errors
	log.Printf("[Setup] Uninstalling existing %s (if any)...", packageName)
	uninstallCmd := exec.Command("npm", "uninstall", "-g", packageName)
	sysutil.HideWindow(uninstallCmd)
	uninstallCmd.Run() // Ignore errors, package may not exist

	// Clean up leftover temp directories that cause ENOTEMPTY errors
	cleanupNpmTempDirs(packageName)

	// Install the package
	cmdStr := fmt.Sprintf("npm install -g --registry=%s %s", registry, packageName)
	log.Printf("[Setup] Installing global package: %s", packageName)
	log.Printf("[Setup] Command: %s", cmdStr)
	logFn(fmt.Sprintf("Running: %s", cmdStr))

	cmd := exec.Command("npm", "install", "-g", "--registry="+registry, packageName)
	sysutil.HideWindow(cmd)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	if outputStr != "" {
		for _, line := range strings.Split(outputStr, "\n") {
			if line = strings.TrimSpace(line); line != "" {
				log.Printf("[Setup]   %s", line)
			}
		}
	}

	if err != nil {
		log.Printf("[Setup] Failed to install %s: %v", packageName, err)
		if strings.Contains(outputStr, "npm ERR!") {
			return fmt.Errorf("failed to install: %s", strings.TrimSpace(outputStr))
		}
		return fmt.Errorf("install failed: %w", err)
	}

	log.Printf("[Setup] Successfully installed %s", packageName)
	logFn("Installation completed")
	return nil
}

// cleanupNpmTempDirs removes leftover npm temp directories that cause ENOTEMPTY errors
func cleanupNpmTempDirs(packageName string) {
	// Get npm global prefix
	cmd := exec.Command("npm", "config", "get", "prefix")
	sysutil.HideWindow(cmd)
	output, err := cmd.Output()
	if err != nil {
		return
	}

	prefix := strings.TrimSpace(string(output))
	nodeModulesPath := filepath.Join(prefix, "lib", "node_modules")

	// Parse package scope and name (e.g., "@anthropic-ai/claude-code" -> scope="@anthropic-ai", name="claude-code")
	parts := strings.Split(packageName, "/")
	if len(parts) != 2 || !strings.HasPrefix(parts[0], "@") {
		return
	}

	scope := parts[0]
	name := parts[1]
	scopeDir := filepath.Join(nodeModulesPath, scope)

	// Find and remove temp directories like ".claude-code-2DTsDk1V"
	entries, err := os.ReadDir(scopeDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		entryName := entry.Name()
		// Match patterns like ".claude-code-xxxxx" or "claude-code"
		if strings.HasPrefix(entryName, "."+name+"-") || entryName == name {
			targetPath := filepath.Join(scopeDir, entryName)
			log.Printf("[Setup] Cleaning up: %s", targetPath)
			os.RemoveAll(targetPath)
		}
	}
}
