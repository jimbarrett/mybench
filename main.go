package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"mybench/internal/api"
	"mybench/internal/database"
	"mybench/internal/store"
	"mybench/internal/update"

	"github.com/pkg/browser"
)

var version = "dev"

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "version":
			cmdVersion()
			return
		case "update":
			cmdUpdate()
			return
		}
	}

	// Default: start the web server
	port := "8080"
	openBrowser := true
	for _, arg := range os.Args[1:] {
		switch arg {
		case "--no-browser":
			openBrowser = false
		default:
			port = arg
		}
	}

	if err := cmdServe(port, openBrowser); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func cmdServe(port string, openBrowser bool) error {
	s, err := store.New()
	if err != nil {
		return fmt.Errorf("failed to open store: %w", err)
	}

	connMgr := database.NewManager()
	h := api.NewHandlers(version, s, connMgr)
	defer h.Shutdown()

	if openBrowser {
		url := fmt.Sprintf("http://localhost:%s", port)
		if err := browser.OpenURL(url); err != nil {
			fmt.Fprintf(os.Stderr, "Could not open browser: %v\n", err)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	return api.StartServer(ctx, h, port)
}

func cmdVersion() {
	fmt.Printf("mybench %s\n", version)

	info, err := update.Check(version)
	if err != nil {
		fmt.Printf("Could not check for updates: %v\n", err)
		return
	}

	if info.UpdateAvailable {
		fmt.Printf("Update available: %s (released %s)\n", info.LatestVersion, info.PublishedAt.Format("2006-01-02"))
		fmt.Printf("Run 'mybench update' to install it.\n")
	} else {
		fmt.Println("You are running the latest version.")
	}
}

func cmdUpdate() {
	fmt.Println("Checking for updates...")

	info, err := update.Check(version)
	if err != nil {
		fmt.Printf("Error checking for updates: %v\n", err)
		os.Exit(1)
	}

	if !info.UpdateAvailable {
		fmt.Printf("Already up to date (%s).\n", version)
		return
	}

	fmt.Printf("Updating %s → %s\n", version, info.LatestVersion)

	if info.DownloadURL == "" {
		fmt.Printf("No binary found for your platform (%s).\n", update.AssetName())
		fmt.Printf("Visit %s to download manually.\n", info.ReleaseURL)
		os.Exit(1)
	}

	binaryPath, canWrite := update.CanWriteBinary()
	if !canWrite {
		fmt.Println("Cannot write to binary location. Try:")
		fmt.Printf("  %s\n", update.ManualUpdateCommand(info.DownloadURL, binaryPath))
		os.Exit(1)
	}

	fmt.Printf("Downloading %s...\n", info.DownloadURL)
	if err := update.Apply(info.DownloadURL); err != nil {
		fmt.Printf("Update failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Updated to %s successfully.\n", info.LatestVersion)
}
