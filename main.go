package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"mybench/internal/api"
	"mybench/internal/database"
	"mybench/internal/store"
	"mybench/internal/update"

	"github.com/pkg/browser"
)

const defaultPort = 10200

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		// No args: default to "start"
		if err := cmdStart("", true); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	switch os.Args[1] {
	case "start":
		port, openBrowser := parseServeArgs(os.Args[2:])
		if err := cmdStart(port, openBrowser); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "stop":
		if err := cmdStop(); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "_serve":
		port, openBrowser := parseServeArgs(os.Args[2:])
		if err := cmdServe(port, openBrowser); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}

	case "version":
		cmdVersion()

	case "update":
		cmdUpdate()

	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func parseServeArgs(args []string) (port string, openBrowser bool) {
	openBrowser = true
	for _, arg := range args {
		switch arg {
		case "--no-browser":
			openBrowser = false
		default:
			port = arg
		}
	}
	return
}

// cmdStart checks if an instance is already running; if so, opens the browser.
// Otherwise it forks a background _serve process.
func cmdStart(port string, openBrowser bool) error {
	pid, runningPort, alive := readPidFile()
	if alive {
		fmt.Printf("mybench is already running (pid %d) at http://localhost:%s\n", pid, runningPort)
		if openBrowser && runningPort != "" {
			browser.OpenURL(fmt.Sprintf("http://localhost:%s", runningPort))
		}
		return nil
	}

	// Build the _serve command with the same args
	args := []string{"_serve"}
	if port != "" {
		args = append(args, port)
	}
	if !openBrowser {
		args = append(args, "--no-browser")
	}

	// Ensure data dir exists for the log file
	dir := dataDir()
	os.MkdirAll(dir, 0755)

	logPath := filepath.Join(dir, "mybench.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}

	cmd := exec.Command(os.Args[0], args...)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	// Detach from parent process group so it survives terminal close
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("starting background process: %w", err)
	}
	logFile.Close()

	// Wait briefly for the child to write the PID file with the port
	var startedPort string
	for i := 0; i < 20; i++ {
		time.Sleep(100 * time.Millisecond)
		if _, p, alive := readPidFile(); alive && p != "" {
			startedPort = p
			break
		}
	}

	if startedPort != "" {
		fmt.Printf("mybench started at \033[1mhttp://localhost:%s\033[0m (pid %d)\n", startedPort, cmd.Process.Pid)
	} else {
		fmt.Printf("mybench started (pid %d)\n", cmd.Process.Pid)
	}
	fmt.Printf("Log file: %s\n", logPath)
	return nil
}

// cmdStop reads the PID file and sends SIGTERM to the running process.
func cmdStop() error {
	pid, _, alive := readPidFile()
	if !alive {
		fmt.Println("mybench is not running")
		return nil
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process: %w", err)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		os.Remove(pidFilePath())
		fmt.Println("mybench stopped")
		return nil
	}

	// Wait for the process to exit (up to 5 seconds)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if err := proc.Signal(syscall.Signal(0)); err != nil {
			os.Remove(pidFilePath())
			fmt.Println("mybench stopped")
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Force kill if still alive
	_ = proc.Kill()
	os.Remove(pidFilePath())
	fmt.Println("mybench stopped (forced)")
	return nil
}

// cmdServe is the internal foreground server command invoked by start.
func cmdServe(port string, openBrowser bool) error {
	if port == "" {
		found, err := findAvailablePort(defaultPort)
		if err != nil {
			return fmt.Errorf("finding available port: %w", err)
		}
		port = fmt.Sprintf("%d", found)
	}

	// Write PID file
	pidContent := fmt.Sprintf("%d:%s", os.Getpid(), port)
	if err := os.WriteFile(pidFilePath(), []byte(pidContent), 0644); err != nil {
		return fmt.Errorf("writing pid file: %w", err)
	}
	defer os.Remove(pidFilePath())

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

	// Set up context that cancels on SIGTERM/SIGINT
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

// --- PID file helpers ---

func dataDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".config", "mybench")
}

func pidFilePath() string {
	return filepath.Join(dataDir(), "mybench.pid")
}

func readPidFile() (int, string, bool) {
	data, err := os.ReadFile(pidFilePath())
	if err != nil {
		return 0, "", false
	}

	parts := strings.SplitN(strings.TrimSpace(string(data)), ":", 2)
	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, "", false
	}

	port := ""
	if len(parts) > 1 {
		port = parts[1]
	}

	// Check if process is alive by sending signal 0
	proc, err := os.FindProcess(pid)
	if err != nil {
		return pid, port, false
	}
	if err := proc.Signal(syscall.Signal(0)); err != nil {
		os.Remove(pidFilePath())
		return pid, port, false
	}

	return pid, port, true
}

// --- Port discovery ---

func findAvailablePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ {
		ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
		if err != nil {
			continue
		}
		ln.Close()
		return port, nil
	}
	return 0, fmt.Errorf("no available port found in range %d-%d", startPort, startPort+99)
}

// --- Version / Update ---

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

func printUsage() {
	fmt.Fprintf(os.Stderr, `Usage: mybench <command>

Commands:
  start [port] [--no-browser]  Start mybench in the background (default)
  stop                         Stop the running instance
  version                      Show version and check for updates
  update                       Update to the latest version
`)
}
