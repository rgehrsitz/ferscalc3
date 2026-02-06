package cli

import (
	"fmt"
	"log"
	"os/exec"
	"runtime"
	"time"

	"github.com/rpgo/retirement-calculator/internal/web"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web UI server",
	Long: `Start the FERS Retirement Calculator web server with an embedded frontend.

The server serves both the API endpoints and a single-page web application
that provides a guided wizard for configuring and running retirement calculations.

By default, the server opens your browser automatically.

Examples:
  fers-calc serve
  fers-calc serve --port 3000
  fers-calc serve --no-open`,
	Run: runServe,
}

func init() {
	serveCmd.Flags().StringP("port", "p", "8080", "Port to listen on")
	serveCmd.Flags().Bool("no-open", false, "Don't auto-open the browser")
	serveCmd.Flags().Bool("no-idle-shutdown", false, "Don't auto-exit when the browser is closed")
	rootCmd.AddCommand(serveCmd)
}

// runServe is the shared implementation for both the root command (no subcommand)
// and the explicit "serve" subcommand. Either way, it starts the web server and
// auto-opens the browser.
func runServe(cmd *cobra.Command, args []string) {
	port, _ := cmd.Flags().GetString("port")
	noOpen, _ := cmd.Flags().GetBool("no-open")
	noIdleShutdown, _ := cmd.Flags().GetBool("no-idle-shutdown")

	server := web.NewServer(port)

	// Disable idle-shutdown if requested
	if noIdleShutdown {
		server.SetIdleShutdown(false)
	}

	// Auto-open browser unless --no-open is set
	if !noOpen {
		go func() {
			time.Sleep(500 * time.Millisecond)
			url := fmt.Sprintf("http://localhost:%s", port)
			if err := openBrowser(url); err != nil {
				log.Printf("Could not open browser: %v", err)
				log.Printf("Open %s in your browser manually", url)
			}
		}()
	}

	if err := server.Run(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// openBrowser opens the specified URL in the default browser.
// Uses platform-specific commands: open (macOS), xdg-open (Linux), rundll32 (Windows).
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	return cmd.Start()
}
