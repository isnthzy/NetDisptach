package egress

import (
	"sync"

	"netdispatch/internal/router"
)

// ProxyConfig represents upstream proxy configuration
type ProxyConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // "http" or "socks5"
	Username string `json:"username"`
	Password string `json:"password"`
}

// Policy represents an egress policy
type Policy struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	NIC         string       `json:"nic"`
	Proxy       *ProxyConfig `json:"proxy"`
	Description string       `json:"description"`
}

// Manager manages egress policies
type Manager struct {
	mu            sync.RWMutex
	routerMgr     *router.Manager
	policies      map[string]*Policy
	defaultPolicy string
}

// NewManager creates a new egress manager
func NewManager() *Manager {
	return &Manager{
		routerMgr: router.NewManager(),
		policies:  make(map[string]*Policy),
	}
}

// SetRouterManager sets the router manager
func (m *Manager) SetRouterManager(rm *router.Manager) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.routerMgr = rm
}

// RouterManager returns the router manager
func (m *Manager) RouterManager() *router.Manager {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.routerMgr
}

// Add adds a new egress policy
func (m *Manager) Add(policy *Policy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies[policy.ID] = policy
}

// Remove removes an egress policy
func (m *Manager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.policies, id)
}

// Get returns an egress policy by ID
func (m *Manager) Get(id string) *Policy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.policies[id]
}

// List returns all egress policies
func (m *Manager) List() []*Policy {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*Policy, 0, len(m.policies))
	for _, p := range m.policies {
		result = append(result, p)
	}
	return result
}

// SetDefault sets the default egress policy
func (m *Manager) SetDefault(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.defaultPolicy = id
}

// Default returns the default egress policy
func (m *Manager) Default() *Policy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.policies[m.defaultPolicy]
}

// SetPolicies replaces all policies with the provided list
func (m *Manager) SetPolicies(policies []*Policy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.policies = make(map[string]*Policy)
	for _, p := range policies {
		if p != nil {
			m.policies[p.ID] = p
		}
	}
}

// Select selects an egress policy for a target address
func (m *Manager) Select(targetAddr string) (*Policy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Helper to find a policy by ID with fallback to default, then first available
	findPolicy := func(id string) *Policy {
		if id != "" {
			if p, ok := m.policies[id]; ok && p != nil {
				return p
			}
		}
		if p, ok := m.policies[m.defaultPolicy]; ok && p != nil {
			return p
		}
		for _, p := range m.policies {
			if p != nil {
				return p
			}
		}
		return nil
	}

	// If no router manager, use default policy
	if m.routerMgr == nil {
		if p := findPolicy(""); p != nil {
			return p, nil
		}
		return nil, ErrNoPolicyAvailable
	}

	// Use router to make routing decision
	decision := m.routerMgr.Route(targetAddr)

	if decision.Action == router.ActionReject {
		return nil, ErrConnectionRejected
	}

	if p := findPolicy(decision.EgressID); p != nil {
		return p, nil
	}
	return nil, ErrNoPolicyAvailable
}
