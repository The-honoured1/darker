package engine

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"darker/internal/db"
	"darker/pkg/tor"

	"golang.org/x/net/html"
)

var onionRegex = regexp.MustCompile(`[a-z2-7]{16,56}\.onion`)

// Crawler handles the discovery and scraping of onion services.
type Crawler struct {
	torClient *tor.Client
	store     *db.Store
	queue     chan string
	visited   sync.Map
}

// NewCrawler creates a new crawler instance.
func NewCrawler(torClient *tor.Client, store *db.Store) *Crawler {
	return &Crawler{
		torClient: torClient,
		store:     store,
		queue:     make(chan string, 1000),
	}
}

// Start runs the crawler with a set of seed URLs and loads pending sites from DB.
func (c *Crawler) Start(ctx context.Context, seeds []string) {
	// Load unscanned sites from database to resume work
	unscanned, err := c.store.GetUnscannedSites(100)
	if err == nil {
		for _, url := range unscanned {
			c.enqueue(url)
		}
	}

	for _, seed := range seeds {
		c.enqueue(seed)
	}

	// Simple worker pool
	for i := 0; i < 5; i++ {
		go c.worker(ctx)
	}

	// Periodically look for more work in the DB
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				unscanned, err := c.store.GetUnscannedSites(50)
				if err == nil {
					for _, url := range unscanned {
						c.enqueue(url)
					}
				}
			}
		}
	}()
}

func (c *Crawler) enqueue(url string) {
	// Clean the URL/onion address
	url = strings.TrimSpace(url)
	if url == "" {
		return
	}

	// If it's just an onion address, add schema
	if onionRegex.MatchString(url) && !strings.HasPrefix(url, "http") {
		url = "http://" + url
	}

	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return // Ignore non-http links for now
	}

	if _, loaded := c.visited.LoadOrStore(url, true); !loaded {
		select {
		case c.queue <- url:
		default:
			// Queue full, skip for now
		}
	}
}

func (c *Crawler) worker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case url := <-c.queue:
			if err := c.scrape(url); err != nil {
				log.Printf("Failed to scrape %s: %v", url, err)
			}
			// Be polite to the network
			time.Sleep(1 * time.Second)
		}
	}
}

func (c *Crawler) scrape(targetURL string) error {
	resp, err := c.torClient.Get(targetURL)
	if err != nil {
		// Log failure but ensure it's in DB so we can retry later
		c.store.UpsertSite(&db.Site{
			URL:      targetURL,
			LastSeen: time.Now(),
			IsActive: false,
		})
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return err
	}

	site := &db.Site{
		URL:      targetURL,
		LastSeen: time.Now(),
		IsActive: true,
	}

	c.extractMetadata(doc, site)
	c.store.UpsertSite(site)
	c.discoverLinks(doc, targetURL)

	return nil
}

func (c *Crawler) extractMetadata(n *html.Node, s *db.Site) {
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "title" && n.FirstChild != nil {
			s.Title = strings.TrimSpace(n.FirstChild.Data)
		}
		if n.Type == html.ElementNode && n.Data == "meta" {
			var name, content string
			for _, attr := range n.Attr {
				if attr.Key == "name" {
					name = attr.Val
				}
				if attr.Key == "content" {
					content = attr.Val
				}
			}
			if strings.ToLower(name) == "description" {
				s.Description = content
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
}

func (c *Crawler) discoverLinks(n *html.Node, baseURL string) {
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					link := attr.Val
					// Absolute onion address search
					matches := onionRegex.FindString(link)
					if matches != "" {
						c.enqueue(matches)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
}
