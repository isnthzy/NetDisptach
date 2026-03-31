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

	// If no router manager, use default policy
	if m.routerMgr == nil {
		if policy, ok := m.policies[m.defaultPolicy]; ok && policy != nil {
			return policy, nil
		}
		// Return first available policy as fallback
		for _, policy := range m.policies {
			if policy != nil {
				return policy, nil
			}
		}
		return nil, ErrNoPolicyAvailable
	}

	// Use routerMgr's Route method directly
	decision := m.routerMgr.Route(targetAddr)

	switch decision.Action {
	case router.ActionReject:
		return nil, ErrConnectionRejected
	case router.ActionForward:
		if decision.EgressID != "" {
			if policy, ok := m.policies[decision.EgressID]; ok && policy != nil {
				return policy, nil
			}
		}
		// Fallback to default policy
		if policy, ok := m.policies[m.defaultPolicy]; ok && policy != nil {
			return policy, nil
		}
		// Return first available policy as fallback
		for _, policy := range m.policies {
			if policy != nil {
				return policy, nil
			}
		}
		return nil, ErrNoPolicyAvailable
	default:
		if policy, ok := m.policies[m.defaultPolicy]; ok && policy != nil {
			return policy, nil
		}
		// Return first available policy as fallback
		for _, policy := range m.policies {
			if policy != nil {
				return policy, nil
			}
		}
		return nil, ErrNoPolicyAvailable
	}
}
