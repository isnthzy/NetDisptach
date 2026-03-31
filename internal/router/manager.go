package router

import (
	"sort"
	"sync"
)

// Manager manages routing rules
type Manager struct {
	mu           sync.RWMutex
	rules        []Rule
	defaultEgress string
}

// NewManager creates a new router manager
func NewManager() *Manager {
	return &Manager{
		rules: make([]Rule, 0),
	}
}

// AddRule adds a routing rule
func (m *Manager) AddRule(rule Rule) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, r := range m.rules {
		if r.ID == rule.ID {
			m.rules[i] = rule
			return
		}
	}

	m.rules = append(m.rules, rule)
	m.sortRules()
}

// RemoveRule removes a routing rule
func (m *Manager) RemoveRule(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, r := range m.rules {
		if r.ID == id {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)
			return
		}
	}
}

// ListRules returns all rules
func (m *Manager) ListRules() []Rule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]Rule, len(m.rules))
	copy(result, m.rules)
	return result
}

// GetRule returns a rule by ID
func (m *Manager) GetRule(id string) *Rule {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, r := range m.rules {
		if r.ID == id {
			return &r
		}
	}
	return nil
}

// SetDefaultEgress sets the default egress policy
func (m *Manager) SetDefaultEgress(egressID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultEgress = egressID
}

// GetDefaultEgress returns the default egress policy ID
func (m *Manager) GetDefaultEgress() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultEgress
}

// SetRules replaces all rules with the provided list
func (m *Manager) SetRules(rules []Rule) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = make([]Rule, len(rules))
	copy(m.rules, rules)
	m.sortRules()
}

// ClearRules removes all rules
func (m *Manager) ClearRules() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rules = make([]Rule, 0)
}

func (m *Manager) sortRules() {
	sort.Slice(m.rules, func(i, j int) bool {
		return m.rules[i].Priority < m.rules[j].Priority
	})
}

// Route makes a routing decision for a target address
func (m *Manager) Route(targetAddr string) Decision {
	m.mu.RLock()
	defer m.mu.RUnlock()

	host, port := parseAddr(targetAddr)

	for _, rule := range m.rules {
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
		EgressID: m.defaultEgress,
	}
}
