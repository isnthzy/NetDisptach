package nic

import (
	"net"
	"sync"
)

// NIC represents a network interface card
type NIC struct {
	Name        string   `json:"name"`
	DisplayName string   `json:"display_name"`
	IP          string   `json:"ip"`
	Netmask     string   `json:"netmask"`
	MAC         string   `json:"mac"`
	IsUp        bool     `json:"is_up"`
	IsDefault   bool     `json:"is_default"`
	Flags       []string `json:"flags"`
}

// Manager manages network interfaces
type Manager struct {
	mu         sync.RWMutex
	interfaces []NIC
	defaultNIC string
}

// NewManager creates a new NIC manager
func NewManager() *Manager {
	return &Manager{
		interfaces: make([]NIC, 0),
	}
}

// Refresh refreshes the list of network interfaces
func (m *Manager) Refresh() error {
	ifaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	newInterfaces := make([]NIC, 0)

	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			if ipNet.IP.To4() == nil {
				continue
			}

			flags := make([]string, 0)
			if iface.Flags&net.FlagUp != 0 {
				flags = append(flags, "up")
			}
			if iface.Flags&net.FlagBroadcast != 0 {
				flags = append(flags, "broadcast")
			}
			if iface.Flags&net.FlagMulticast != 0 {
				flags = append(flags, "multicast")
			}

			nic := NIC{
				Name:        iface.Name,
				DisplayName: iface.Name,
				IP:          ipNet.IP.String(),
				Netmask:     net.IP(ipNet.Mask).String(),
				MAC:         iface.HardwareAddr.String(),
				IsUp:        iface.Flags&net.FlagUp != 0,
				Flags:       flags,
			}

			newInterfaces = append(newInterfaces, nic)
		}
	}

	m.mu.Lock()
	m.interfaces = newInterfaces
	m.mu.Unlock()

	m.detectDefaultNIC()
	return nil
}

// detectDefaultNIC detects the default NIC based on routing
func (m *Manager) detectDefaultNIC() {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return
	}
	defer conn.Close()

	localAddr := conn.LocalAddr()
	if localAddr == nil {
		return
	}

	udpAddr, ok := localAddr.(*net.UDPAddr)
	if !ok {
		return
	}
	localIP := udpAddr.IP.String()

	m.mu.Lock()
	defer m.mu.Unlock()

	for i := range m.interfaces {
		if m.interfaces[i].IP == localIP {
			m.interfaces[i].IsDefault = true
			m.defaultNIC = m.interfaces[i].Name
			break
		}
	}
}

// List returns all detected NICs
func (m *Manager) List() []NIC {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]NIC, len(m.interfaces))
	copy(result, m.interfaces)
	return result
}

// Get returns a NIC by name
func (m *Manager) Get(name string) *NIC {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for i := range m.interfaces {
		if m.interfaces[i].Name == name {
			return &m.interfaces[i]
		}
	}
	return nil
}

// Default returns the default NIC
func (m *Manager) Default() *NIC {
	m.mu.RLock()
	defaultName := m.defaultNIC
	m.mu.RUnlock()
	return m.Get(defaultName)
}

// GetIP returns the IP address of a NIC
func (m *Manager) GetIP(name string) net.IP {
	nic := m.Get(name)
	if nic == nil {
		return nil
	}
	return net.ParseIP(nic.IP)
}
