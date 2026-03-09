package main

import (
	"embed"
	"fmt"
	"os"

	"mybench/internal/update"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

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

	app := NewApp(version)

	err := wails.Run(&options.App{
		Title:     "mybench",
		Width:     1280,
		Height:    800,
		MinWidth:  900,
		MinHeight: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		// Tokyo Night --bg-primary: #1a1b26
		BackgroundColour: &options.RGBA{R: 26, G: 27, B: 38, A: 255},
		OnStartup:        app.startup,
		OnShutdown:       app.shutdown,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		println("Error:", err.Error())
	}
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
