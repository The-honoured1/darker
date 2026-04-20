package main

import (
	"context"
	"log"
	"os"
	"time"

	"darker/internal/db"
	"darker/internal/engine"
	"darker/internal/ui"
	"darker/pkg/tor"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	// Initialize Database
	store, err := db.NewStore("darker.db")
	if err != nil {
		log.Fatalf("Critical: Failed to initialize database: %v\nNote: If you see 'no such module: fts5', run with: go run -tags \"fts5\" main.go", err)
	}
	defer store.Close()

	// Try to connect to existing Tor proxy first
	torClient, err := tor.NewClient("127.0.0.1:9050")
	var torManager *tor.Manager

	if err != nil || torClient.CheckConnection() != nil {
		log.Println("Existing Tor proxy not found or unreachable. Attempting to start local Tor...")
		torManager = &tor.Manager{}
		if err := torManager.Start(); err != nil {
			log.Printf("Failed to start local Tor: %v", err)
			log.Println("Ensure Tor is installed ('sudo apt install tor') and accessible.")
		} else {
			log.Println("Local Tor process started. Waiting for connection...")
			// Wait for Tor to bootstrap
			// We retry connection check for a bit
			connected := false
			for i := 0; i < 30; i++ {
				if torClient.CheckConnection() == nil {
					connected = true
					break
				}
				time.Sleep(2 * time.Second)
			}
			if !connected {
				log.Println("Warning: Local Tor started but connectivity check still failing.")
			} else {
				log.Println("Connected to local Tor successfully.")
			}
		}
	} else {
		log.Println("Connected to existing Tor proxy.")
	}

	if torManager != nil {
		defer torManager.Stop()
	}

	// Initialize Crawler
	crawler := engine.NewCrawler(torClient, store)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start background crawler with some default seeds
	seeds := []string{
		"http://zqktlwiuavvvqqt4ybvgvi7tyo4hjlbt674h7sy33yxpw73t3uhl7ad.onion", // Hidden Wiki
		"http://darkwebno911.onion",
		"http://torlinks.onion",
	}
	go crawler.Start(ctx, seeds)

	// Run TUI
	p := tea.NewProgram(ui.InitialModel(store), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Printf("Error running TUI: %v", err)
		os.Exit(1)
	}
}
