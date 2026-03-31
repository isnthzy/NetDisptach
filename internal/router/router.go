package router

import (
	"fmt"
	"net"
	"sort"
	"sync"
)

// ActionType represents the action to take
type ActionType int

const (
	ActionForward ActionType = iota
	ActionReject
)

// Decision represents a routing decision
type Decision struct {
	Action   ActionType
	EgressID string
	RuleID   string
}

// Router handles routing decisions
type Router struct {
	mu            sync.RWMutex
	rules         []Rule
	defaultEgress string
}

// NewRouter creates a new router
func NewRouter() *Router {
	return &Router{
		rules: make([]Rule, 0),
	}
}

// AddRule adds a routing rule
func (r *Router) AddRule(rule Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = append(r.rules, rule)
	sort.Slice(r.rules, func(i, j int) bool {
		return r.rules[i].Priority < r.rules[j].Priority
	})
}

// RemoveRule removes a routing rule
func (r *Router) RemoveRule(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, rule := range r.rules {
		if rule.ID == id {
			r.rules = append(r.rules[:i], r.rules[i+1:]...)
			break
		}
	}
}

// SetRules sets all rules
func (r *Router) SetRules(rules []Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rules = rules
	sort.Slice(r.rules, func(i, j int) bool {
		return r.rules[i].Priority < r.rules[j].Priority
	})
}

// SetDefaultEgress sets the default egress policy
func (r *Router) SetDefaultEgress(egressID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaultEgress = egressID
}

// Route makes a routing decision for a target address
func (r *Router) Route(targetAddr string) Decision {
	r.mu.RLock()
	defer r.mu.RUnlock()

	host, port := parseAddr(targetAddr)

	for _, rule := range r.rules {
		if !rule.Enabled {
			continue
		}

		if rule.Match(host, port) {
			action := ActionForward
			if rule.Action == "reject" {
				action = ActionReject
			}
			return Decision{
				Action:   action,
				EgressID: rule.EgressID,
				RuleID:   rule.ID,
			}
		}
	}

	return Decision{
		Action:   ActionForward,
		EgressID: r.defaultEgress,
	}
}

func parseAddr(addr string) (host string, port int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 0
	}
	port = 0
	if portStr != "" {
		fmt.Sscanf(portStr, "%d", &port)
	}
	return host, port
}
