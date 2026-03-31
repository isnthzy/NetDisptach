package router

import (
	"net"
)

// ListType represents the type of list (whitelist or blacklist)
type ListType string

const (
	ListTypeNone      ListType = "none"       // Normal rule
	ListTypeWhitelist ListType = "whitelist"  // Whitelist - only these are allowed
	ListTypeBlacklist ListType = "blacklist"  // Blacklist - these are blocked
)

// Rule represents a routing rule
type Rule struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Priority    int      `json:"priority"`
	Enabled     bool     `json:"enabled"`
	ListType    ListType `json:"list_type"`    // "none", "whitelist", "blacklist"
	Domains     []string `json:"domains"`
	CIDRs       []string `json:"cidrs"`
	Ports       []int    `json:"ports"`
	Action      string   `json:"action"` // "forward" or "reject"
	EgressID    string   `json:"egress_id"`
	Description string   `json:"description"`

	// Fields for imported domain lists
	Source      string `json:"source,omitempty"`       // Import source (URL or file path)
	DomainCount int    `json:"domain_count,omitempty"` // Number of domains in the list

	// Internal fields
	cidrNets   []*net.IPNet
	domainTree *DomainTree  // Optimized tree for large domain lists
	portMap    map[int]bool // Compiled port set for O(1) lookup
}

// Match checks if the target matches this rule
// All specified conditions (Domains, CIDRs, Ports) must match (AND logic)
func (r *Rule) Match(host string, port int) bool {
	// If domains are specified, host must match one of them
	if len(r.Domains) > 0 || r.domainTree != nil {
		matched := false

		// Prefer optimized tree structure for large domain lists
		if r.domainTree != nil {
			matched = r.domainTree.Match(host)
		} else {
			// Fallback to linear search for small lists
			for _, domain := range r.Domains {
				if matchDomain(host, domain) {
					matched = true
					break
				}
			}
		}
		if !matched {
			return false
		}
	}

	// If CIDRs are specified, IP must match one of them
	if len(r.cidrNets) > 0 {
		ip := net.ParseIP(host)
		if ip == nil {
			return false // Host is not an IP but CIDRs are specified
		}
		matched := false
		for _, cidr := range r.cidrNets {
			if cidr.Contains(ip) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// If ports are specified, port must match one of them
	if len(r.Ports) > 0 {
		if r.portMap != nil {
			if !r.portMap[port] {
				return false
			}
		} else {
			// Fallback for uncompiled rules
			matched := false
			for _, p := range r.Ports {
				if p == port {
					matched = true
					break
				}
			}
			if !matched {
				return false
			}
		}
	}

	// If no conditions are specified, rule matches everything
	return len(r.Domains) > 0 || len(r.cidrNets) > 0 || len(r.Ports) > 0
}

// CompileCIDRs compiles CIDR strings to net.IPNet
func (r *Rule) CompileCIDRs() error {
	r.cidrNets = make([]*net.IPNet, 0, len(r.CIDRs))
	for _, cidr := range r.CIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return err
		}
		r.cidrNets = append(r.cidrNets, ipNet)
	}
	// Also compile port map
	if len(r.Ports) > 0 {
		r.portMap = make(map[int]bool, len(r.Ports))
		for _, p := range r.Ports {
			r.portMap[p] = true
		}
	}
	return nil
}

// MatchDomain checks if a domain matches a pattern
func matchDomain(target, pattern string) bool {
	if pattern == "*" {
		return true
	}

	// Handle wildcard patterns like "*.example.com"
	if len(pattern) >= 2 && pattern[0] == '*' && pattern[1] == '.' {
		suffix := pattern[1:] // ".example.com"
		// Target should end with the suffix and be longer than the suffix
		return len(target) > len(suffix) && target[len(target)-len(suffix):] == suffix
	}

	return target == pattern
}

// BuildDomainTree builds an optimized domain tree from the Domains slice.
// This should be called after loading a rule with many domains for efficient matching.
func (r *Rule) BuildDomainTree() {
	if len(r.Domains) == 0 {
		return
	}

	r.domainTree = NewDomainTree()
	r.domainTree.AddMultiple(r.Domains)
}

// ClearDomainTree removes the domain tree to free memory.
func (r *Rule) ClearDomainTree() {
	r.domainTree = nil
}

// HasDomainTree returns true if the rule has an optimized domain tree.
func (r *Rule) HasDomainTree() bool {
	return r.domainTree != nil
}
