package main

import (
	"log"
	"os"

	"github.com/rpgo/retirement-calculator/internal/web"
	"github.com/spf13/cobra"
)

func main() {
	var port string

	rootCmd := &cobra.Command{
		Use:   "web-server",
		Short: "FERS Calculator Web Server",
		Long:  "Web server for the FERS Calculator application, providing API endpoints for retirement planning calculations",
		Run: func(cmd *cobra.Command, args []string) {
			server := web.NewServer(port)
			if err := server.Run(); err != nil {
				log.Fatalf("Server failed to run: %v", err)
			}
		},
	}

	rootCmd.PersistentFlags().StringVarP(&port, "port", "p", "8080", "Port to listen on")

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}
