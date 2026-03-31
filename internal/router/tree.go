package router

import (
	"strings"
	"sync"
)

// DomainTree is a domain suffix tree for efficient domain matching.
// It stores domains in reverse order (by label) to support efficient suffix matching.
// For example, "google.com" is stored as ["com", "google"] and will match
// both "google.com" and any subdomain like "www.google.com".
type DomainTree struct {
	root  *domainNode
	mu    sync.RWMutex
	count int
}

type domainNode struct {
	children map[string]*domainNode
	isEnd    bool // marks the end of a domain
}

// NewDomainTree creates a new empty domain tree.
func NewDomainTree() *DomainTree {
	return &DomainTree{
		root: &domainNode{
			children: make(map[string]*domainNode),
		},
	}
}

// Add adds a domain to the tree.
// The domain will match itself and all its subdomains.
// For example, adding "google.com" will match "google.com", "www.google.com", "mail.google.com", etc.
func (dt *DomainTree) Add(domain string) {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return
	}

	// Remove leading/trailing dots and wildcards
	domain = strings.Trim(domain, ". ")
	if strings.HasPrefix(domain, "*.") {
		domain = domain[2:]
	}
	if domain == "" {
		return
	}

	dt.mu.Lock()
	defer dt.mu.Unlock()

	parts := reverseDomainLabels(domain)
	node := dt.root

	for _, part := range parts {
		if node.children == nil {
			node.children = make(map[string]*domainNode)
		}
		if node.children[part] == nil {
			node.children[part] = &domainNode{
				children: make(map[string]*domainNode),
			}
		}
		node = node.children[part]
	}

	if !node.isEnd {
		node.isEnd = true
		dt.count++
	}
}

// AddMultiple adds multiple domains to the tree.
func (dt *DomainTree) AddMultiple(domains []string) {
	for _, domain := range domains {
		dt.Add(domain)
	}
}

// Match checks if a domain matches any domain in the tree.
// It supports suffix matching: if "google.com" is in the tree,
// then "google.com", "www.google.com", "mail.google.com" will all match.
func (dt *DomainTree) Match(domain string) bool {
	if domain == "" {
		return false
	}

	domain = strings.Trim(domain, ". ")

	dt.mu.RLock()
	defer dt.mu.RUnlock()

	parts := reverseDomainLabels(domain)
	node := dt.root

	for _, part := range parts {
		// If we hit a node that marks the end of a domain, it's a match
		// This handles the suffix matching: "google.com" matches "www.google.com"
		if node.isEnd {
			return true
		}
		if node.children == nil || node.children[part] == nil {
			return false
		}
		node = node.children[part]
	}

	return node.isEnd
}

// MatchExact checks if a domain exactly matches a domain in the tree.
// Unlike Match, this does not perform suffix matching.
func (dt *DomainTree) MatchExact(domain string) bool {
	if domain == "" {
		return false
	}

	domain = strings.Trim(domain, ". ")

	dt.mu.RLock()
	defer dt.mu.RUnlock()

	parts := reverseDomainLabels(domain)
	node := dt.root

	for _, part := range parts {
		if node.children == nil || node.children[part] == nil {
			return false
		}
		node = node.children[part]
	}

	return node.isEnd
}

// Count returns the number of domains in the tree.
func (dt *DomainTree) Count() int {
	dt.mu.RLock()
	defer dt.mu.RUnlock()
	return dt.count
}

// Clear removes all domains from the tree.
func (dt *DomainTree) Clear() {
	dt.mu.Lock()
	defer dt.mu.Unlock()
	dt.root = &domainNode{
		children: make(map[string]*domainNode),
	}
	dt.count = 0
}

// ToList returns all domains in the tree.
// This is mainly for debugging and testing.
func (dt *DomainTree) ToList() []string {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	var domains []string
	var traverse func(node *domainNode, parts []string)

	traverse = func(node *domainNode, parts []string) {
		if node.isEnd {
			// Reverse the parts to get the original domain
			reversed := make([]string, len(parts))
			for i, p := range parts {
				reversed[len(parts)-1-i] = p
			}
			domains = append(domains, strings.Join(reversed, "."))
		}
		for label, child := range node.children {
			traverse(child, append(parts, label))
		}
	}

	traverse(dt.root, nil)
	return domains
}

// reverseDomainLabels splits a domain by '.' and reverses the order of labels.
// For example: "www.google.com" -> ["com", "google", "www"]
func reverseDomainLabels(domain string) []string {
	parts := strings.Split(domain, ".")
	// Reverse in place
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return parts
}
