package router

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

// ImportResult represents the result of a domain list import
type ImportResult struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	DomainCount int    `json:"domain_count"`
	Source      string `json:"source"`
	Status      string `json:"status"`
	Error       string `json:"error,omitempty"`
}

// Importer handles importing domain lists from various sources
type Importer struct {
	manager *Manager
	client  *http.Client
}

// NewImporter creates a new domain list importer
func NewImporter(manager *Manager) *Importer {
	return &Importer{
		manager: manager,
		client: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// ImportFromURL imports a domain list from a remote URL
func (i *Importer) ImportFromURL(url, name, egressID string, priority int, enabled bool) (*ImportResult, error) {
	// Download the domain list
	resp, err := i.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download from URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Parse domains from response body
	domains, err := parseDomains(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse domains: %w", err)
	}

	return i.createImportedRule(name, domains, url, egressID, priority, enabled)
}

// ImportFromFile imports a domain list from a local file
func (i *Importer) ImportFromFile(filePath, name, egressID string, priority int, enabled bool) (*ImportResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Parse domains from file
	domains, err := parseDomains(file)
	if err != nil {
		return nil, fmt.Errorf("failed to parse domains: %w", err)
	}

	return i.createImportedRule(name, domains, filePath, egressID, priority, enabled)
}

// ImportFromReader imports a domain list from an io.Reader
func (i *Importer) ImportFromReader(reader io.Reader, name, source, egressID string, priority int, enabled bool) (*ImportResult, error) {
	domains, err := parseDomains(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to parse domains: %w", err)
	}

	return i.createImportedRule(name, domains, source, egressID, priority, enabled)
}

// createImportedRule creates a rule from imported domains
func (i *Importer) createImportedRule(name string, domains []string, source, egressID string, priority int, enabled bool) (*ImportResult, error) {
	if len(domains) == 0 {
		return nil, fmt.Errorf("no valid domains found")
	}

	// 验证优先级范围
	if priority < MinPriority || priority > MaxPriority {
		return nil, fmt.Errorf("优先级必须在 %d-%d 之间，当前值: %d", MinPriority, MaxPriority, priority)
	}

	// Generate a unique ID
	id := generateImportID()

	// Create the rule
	rule := Rule{
		ID:          id,
		Name:        name,
		Priority:    priority,
		Enabled:     enabled,
		ListType:    ListTypeNone,
		Domains:     domains,
		DomainCount: len(domains),
		Source:      source,
		EgressID:    egressID,
		Action:      "forward",
	}

	// Build the optimized domain tree
	rule.BuildDomainTree()

	// Add to manager
	i.manager.AddRule(rule)

	return &ImportResult{
		ID:          id,
		Name:        name,
		DomainCount: len(domains),
		Source:      source,
		Status:      "success",
	}, nil
}

// parseDomains reads domain names from a reader, one per line
func parseDomains(reader io.Reader) ([]string, error) {
	var domains []string
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle lines with additional content (e.g., "domain.com # comment")
		if idx := strings.Index(line, "#"); idx > 0 {
			line = strings.TrimSpace(line[:idx])
		}

		// Remove leading/trailing dots
		line = strings.Trim(line, ". ")

		// Skip if empty after trimming
		if line == "" {
			continue
		}

		// Skip wildcards that are just "*"
		if line == "*" {
			continue
		}

		// Skip single-label domains (like "com", "org") - not useful for routing
		if !strings.Contains(line, ".") {
			continue
		}

		// Deduplicate
		if !seen[line] {
			seen[line] = true
			domains = append(domains, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return domains, nil
}

// generateImportID generates a unique ID for an imported rule
var importCounter int64

func generateImportID() string {
	counter := atomic.AddInt64(&importCounter, 1)
	return fmt.Sprintf("imported-%s-%d", time.Now().Format("20060102-150405"), counter)
}
