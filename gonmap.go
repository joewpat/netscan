package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

func main() {
	fmt.Println("go nmap")
	ipv4address, err := getIPv4()
	if err != nil {
		fmt.Println(err)
	}
	valid := is_valid_ipv4(ipv4address)
	if valid != true {
		fmt.Println("error: unable to locate valid ipv4 address")
	}
	fmt.Printf("IPv4 LAN address:\t%v\n", ipv4address)
	fmt.Printf("Scanning Subnet:\t%v\n", ipv4address)
}

func getSubnet(ip string) string {

}

//func scan(ip string) []string {
//hosts := []string
//	fmt.Println(getIPv4())
//	return "scan"
//}

func getIPv4() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("no IPV4 address found")
}

func is_valid_ipv4(ip string) bool {

	var result bool

	if ip == "" {
		result = false
		return result
	}

	s := strings.Split(ip, ".")
	for _, v := range s {
		if v[:1] == "0" && len(v) >= 2 {
			result = false
			break
		}
		y, err := strconv.Atoi(v) // convert str to int for comparison
		if y < 0 || y >= 256 {
			result = false
			break
		} else if len(s) != 4 {
			result = false
			break
		} else if err != nil {
			result = false
			break
		} else {
			result = true
		}
	}
	return result

}
