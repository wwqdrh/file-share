package utils

import (
	"net"
	"strings"
)

// IsLoopback checks if an IP address is a loopback address
func IsLoopback(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// GetLoopback returns the loopback address for the specified family
func GetLoopback(family string) string {
	if strings.ToLower(family) == "ipv6" {
		return "::1"
	}
	return "127.0.0.1"
}

// GetIPAddresses returns all IP addresses for a given network interface
func GetIPAddresses(netInterfaceName string, ipFamily string) []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return []string{}
	}

	var addresses []string
	for _, iface := range interfaces {
		if iface.Name != netInterfaceName {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ip := v.IP
				if ipFamily == "ipv4" && ip.To4() != nil && !IsLoopback(ip.String()) {
					addresses = append(addresses, ip.String())
				} else if ipFamily == "ipv6" && ip.To4() == nil && !IsLoopback(ip.String()) {
					addresses = append(addresses, ip.String())
				}
			}
		}
	}
	return addresses
}

// GetNetInterfaceNames returns all network interface names that have IP addresses
func GetNetInterfaceNames(ipFamily string) []string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return []string{}
	}

	var names []string
	for _, iface := range interfaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		hasIP := false
		for _, addr := range addrs {
			switch v := addr.(type) {
			case *net.IPNet:
				ip := v.IP
				if ipFamily == "ipv4" && ip.To4() != nil && !IsLoopback(ip.String()) {
					hasIP = true
					break
				} else if ipFamily == "ipv6" && ip.To4() == nil && !IsLoopback(ip.String()) {
					hasIP = true
					break
				}
			}
		}

		if hasIP {
			// Filter out virtual interfaces
			if !strings.Contains(strings.ToLower(iface.Name), "loopback") &&
				!strings.Contains(strings.ToLower(iface.Name), "vmware") &&
				!strings.Contains(strings.ToLower(iface.Name), "internal") &&
				!strings.Contains(strings.ToLower(iface.Name), "lo") &&
				!strings.Contains(strings.ToLower(iface.Name), "vEthernet") {
				names = append(names, iface.Name)
			}
		}
	}

	if len(names) == 0 {
		// If no filtered names, return all names with IPs
		for _, iface := range interfaces {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			if len(addrs) > 0 {
				names = append(names, iface.Name)
			}
		}
	}

	return names
}

// GetIPAddress returns an IP address for a given index and family
func GetIPAddress(idx int, ipFamily string) string {
	names := GetNetInterfaceNames(ipFamily)
	if len(names) == 0 {
		return GetLoopback(ipFamily)
	}

	idx = idx % len(names)
	ipAddresses := GetIPAddresses(names[idx], ipFamily)
	if len(ipAddresses) > 0 {
		return ipAddresses[0]
	}
	return GetLoopback(ipFamily)
}
