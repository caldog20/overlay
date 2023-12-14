package ipam

import (
	"errors"
	"fmt"
	"go4.org/netipx"
	"net/netip"
	"sync"
)

type IP struct {
	ip       netip.Addr
	clientID uint32
	hostname string
}

type Ipam struct {
	mu     sync.RWMutex
	pool   map[string]IP
	prefix netip.Prefix
}

func NewIpam(prefix string) (*Ipam, error) {
	i := &Ipam{
		pool: make(map[string]IP),
	}

	err := i.SetPrefix(prefix)
	if err != nil {
		return nil, err
	}

	return i, err
}

func (i *Ipam) SetPrefix(p string) error {
	prefix, err := netip.ParsePrefix(p)
	if err != nil {
		return fmt.Errorf("Error parsing prefix: %v", err)
	}

	i.prefix = prefix

	// Pre-reserve network and bcast address
	i.mu.Lock()
	defer i.mu.Unlock()
	i.pool[i.prefix.Addr().String()] = IP{}
	i.pool[netipx.PrefixLastIP(i.prefix).String()] = IP{}

	return nil
}

func (i *Ipam) AllocateIP(clientID uint32) (string, error) {
	if clientID == 0 {
		return "", errors.New("client ID must not be zero when allocating IP")
	}

	i.mu.Lock()
	defer i.mu.Unlock()
	iprange := netipx.RangeOfPrefix(i.prefix)
	for ip := iprange.From(); i.prefix.Contains(ip); ip = ip.Next() {
		ips := ip.String()
		_, found := i.pool[ips]
		if found {
			continue
		}
		i.pool[ips] = IP{
			ip:       ip,
			clientID: clientID,
			//hostname: hostname,
		}
		return ips, nil
	}

	return "", errors.New("ip pool exhausted, failed to get next available IP")
}

func (i *Ipam) DeallocateIP(ip string) error {
	if ip == "" {
		return errors.New("IP must not be nil when deallocating IP")
	}

	i.mu.Lock()
	defer i.mu.Unlock()

	_, found := i.pool[ip]
	if found {
		delete(i.pool, ip)
		return nil
	}

	return errors.New("ip not found in pool, failed to deallocate")
}

func (i *Ipam) WhoIsByIP(clientIP string) (uint32, error) {
	if clientIP == "" {
		return 0, errors.New("client IP must not be nil when searching for client ID")
	}

	i.mu.RLock()
	defer i.mu.RUnlock()

	ip, found := i.pool[clientIP]
	if found {
		return ip.clientID, nil
	}
	return 0, errors.New("Client IP not found")
}

func (i *Ipam) WhoIsByID(clientID uint32) (string, error) {
	if clientID == 0 {
		return "", errors.New("client ID must not be zero when searching for client IP")
	}

	i.mu.RLock()
	defer i.mu.RUnlock()

	for k, v := range i.pool {
		if v.clientID == clientID {
			return k, nil
		}
	}

	return "", errors.New("Client ID not found")
}
