package main

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/joewpat/gonmap/tree/master/go-ping"
	"github.com/mostlygeek/arp"
)

func main() {
	t0 := time.Now()
	fmt.Printf("gonmap - lightweight network scanner utility\n\n")
	ipv4address, err := getIPv4()
	if err != nil {
		panic(err)
	}
	valid := is_valid_ipv4(ipv4address)
	if valid != true {
		fmt.Println("error: unable to locate valid ipv4 address")
	}
	fmt.Printf("IPv4 LAN address:\t%v\n", ipv4address)
	targets := getTargets(ipv4address)
	hostsAvailable := 0
	for _, v := range targets {
		l, r := testconnection(v)
		if l <= 99 { //filter out down clients
			host, _ := net.LookupAddr(v)
			if len(host) == 0 {
				host = append(host, "N/A\t")
			}
			m := arp.Search(v) // get MAC address
			if m == "" {
				m = "N/A\t\t"
			}
			fmt.Println("Host: ", v, "\tRTT: ", r, "\tMAC:\t", m, "\tHostname:", host[0])
			hostsAvailable++
		}
	}
	total := len(targets)
	t1 := time.Now()
	elapsed := t1.Sub(t0)
	d, _ := time.ParseDuration("1s")
	fmt.Printf("\nTargets Scanned: %v\tUp: %v\nTotal Scan Time: %v\n\n", total, hostsAvailable, elapsed.Round(d))
}

/*
testconnection will take a target ipv4 address and return packetloss, rtt
This function can be used to quickly verify a host is responding to ICMP packets
*/
func testconnection(target string) (float64, time.Duration) {
	timeout := time.Millisecond * 100
	pinger, err := ping.NewPinger(target)
	if err != nil {
		panic(err)
	}
	pinger.SetPrivileged(true)
	pinger.Timeout = timeout
	pinger.Count = 1
	err = pinger.Run() // blocks until finished
	if err != nil {
		fmt.Println(err)
	}
	stats := pinger.Statistics() // get send/receive/rtt stats | TODO: get hostnames if possible
	d, _ := time.ParseDuration("0.01ms")
	return stats.PacketLoss, stats.AvgRtt.Round(d)
}
func getTargets(ip string) []string {
	split := strings.Split(ip, ".")
	var iprange []string
	subnet := split[0] + "." + split[1] + "." + split[2] + "." + "0/24"
	fmt.Printf("Scanning subnet:\t%v\n\n", subnet)
	for i := 1; i < 255; i++ {
		fourthoctet := strconv.Itoa(i)
		target := split[0] + "." + split[1] + "." + split[2] + "." + fourthoctet
		iprange = append(iprange, target)
	}
	return iprange
}
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
		y, err := strconv.Atoi(v) // convert str to int for iteration
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

/*TODO
get defaultgateway
concurrency****
*/
