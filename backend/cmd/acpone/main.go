package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/anthropics/acpone/internal/api"
	"github.com/anthropics/acpone/internal/config"
)

//go:embed web/*
var webFS embed.FS

func main() {
	var (
		configPath = flag.String("config", "", "Config file path")
		port       = flag.String("port", "3000", "Server port")
		webDir     = flag.String("web", "", "Web directory (overrides embedded)")
	)
	flag.Parse()

	// Ensure config exists (copy example if needed)
	if err := config.EnsureConfigExists(); err != nil {
		fmt.Printf("⚠️  Config initialization: %v\n", err)
	}

	// Load config
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid config: %v\n", err)
		os.Exit(1)
	}

	// Print startup info
	printStartupInfo(cfg, config.LoadedConfigPath)

	// Determine static files source
	var staticFS fs.FS
	if *webDir != "" {
		staticFS = os.DirFS(*webDir)
		fmt.Printf("   Web directory: %s\n\n", *webDir)
	} else {
		sub, err := fs.Sub(webFS, "web")
		if err == nil {
			staticFS = sub
			fmt.Println("   Web: embedded files\n")
		} else {
			fmt.Println("   Web: no files available\n")
		}
	}

	// Create server
	server := api.NewServer(cfg, staticFS)

	// Graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		server.Shutdown()
		os.Exit(0)
	}()

	// Start server
	printServerBanner(*port)
	addr := ":" + *port
	if err := server.ListenAndServe(addr); err != nil {
		fmt.Fprintf(os.Stderr, "Server error: %v\n", err)
		os.Exit(1)
	}
}

func printStartupInfo(cfg *config.Config, configPath string) {
	fmt.Println("\n📋 Configuration")
	fmt.Println(strings.Repeat("─", 50))
	if configPath != "" {
		fmt.Printf("   Config file: %s\n", configPath)
	} else {
		fmt.Println("   Config file: (using defaults)")
	}
	fmt.Printf("   Default agent: %s\n", cfg.DefaultAgent)
	if cfg.DefaultWorkspace != "" {
		fmt.Printf("   Default workspace: %s\n", cfg.DefaultWorkspace)
	}
	fmt.Println()

	fmt.Println("📦 Agents")
	fmt.Println(strings.Repeat("─", 50))
	for _, agent := range cfg.Agents {
		isDefault := ""
		if agent.ID == cfg.DefaultAgent {
			isDefault = " (default)"
		}
		permission := getPermissionLabel(agent.PermissionMode)
		fmt.Printf("   %s%s\n", agent.Name, isDefault)
		fmt.Printf("     ID: %s\n", agent.ID)
		fmt.Printf("     Permission: %s\n", permission)
		fmt.Printf("     Command: %s %s\n", agent.Command, strings.Join(agent.Args, " "))
		fmt.Println()
	}

	if len(cfg.Workspaces) > 0 {
		fmt.Println("📁 Workspaces")
		fmt.Println(strings.Repeat("─", 50))
		for _, ws := range cfg.Workspaces {
			isDefault := ""
			if ws.ID == cfg.DefaultWorkspace {
				isDefault = " (default)"
			}
			fmt.Printf("   %s%s\n", ws.Name, isDefault)
			fmt.Printf("     Path: %s\n", ws.Path)
			fmt.Println()
		}
	}
}

func getPermissionLabel(mode string) string {
	switch mode {
	case "", "default":
		return "User Confirmation"
	case "bypass":
		return "Auto Approve"
	default:
		return mode
	}
}

func printServerBanner(port string) {
	fmt.Printf(`
╔════════════════════════════════════════════════╗
║           acpone Web Interface                 ║
╠════════════════════════════════════════════════╣
║  Open http://localhost:%s in your browser   ║
║  Press Ctrl+C to stop                          ║
╚════════════════════════════════════════════════╝

`, port)
}
