package proxy

import (
	"fmt"
	"net"
)

// Proxy represents a proxy server with its associated information.
type Proxy struct {
	id    int               // Unique identifier for the proxy.
	addr  *net.TCPAddr      // TCP address of the proxy server.
	infos map[string]string // A map containing additional information about the proxy.
}

// GetAddress returns the proxy's address in the format "ip:port".
func (p *Proxy) GetAddress() string {
	return fmt.Sprintf("%s:%v", p.infos["ipAddress"], p.infos["port"])
}

// GetAnonymityLevel returns the anonymity level of the proxy.
func (p *Proxy) GetAnonymityLevel() string {
	return p.infos["anonymityLevel"]
}

// GetProtocol returns the protocol used by the proxy (e.g., "http" or "https").
func (p *Proxy) GetProtocol() string {
	return p.infos["protocols"]
}

// GetRemoteAddr returns the TCP address of the proxy server.
func (p *Proxy) GetRemoteAddr() (*net.TCPAddr, error) {
	return p.addr, nil
}
