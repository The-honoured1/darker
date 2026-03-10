package main

import (
	"context"
	"log"
	"os"

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
		log.Fatal(err)
	}
	defer store.Close()

	// Initialize Tor Client (Assumes Tor is running on local 9050)
	torClient, err := tor.NewClient("127.0.0.1:9050")
	if err != nil {
		log.Printf("Error: Failed to initialize Tor client: %v", err)
	} else {
		if err := torClient.CheckConnection(); err != nil {
			log.Printf("Warning: Tor proxy found but connectivity check failed: %v", err)
			log.Println("Ensure Tor is correctly configured and has a circuit established.")
		}
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
