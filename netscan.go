package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-ping/ping"
	"github.com/mostlygeek/arp"
)

type Host struct {
	Hostname  string
	IP        string
	RTT       time.Duration
	MAC       string
	Loss      float64
	LastOctet int
}

func main() {
	scan()
}

func scan() {
	t0 := time.Now()
	fmt.Printf("\nnetscan - lightweight network scanner utility\n\n")
	ipv4address, err := getIPv4()
	if err != nil {
		log.Println(err)
		return
	}
	valid := isValidIpv4(ipv4address)
	if !valid {
		fmt.Println("error: unable to locate valid ipv4 address")
	}
	fmt.Printf("IPv4 LAN address:\t%v\n", ipv4address)
	targets, subnet := getTargets(ipv4address)
	fmt.Printf("Scanning subnet:\t%v\n\n", subnet)
	var hosts []Host
	//display output
	fmt.Println("Hostname\t IP Address\t MAC Address\t\t Average RTT")
	fmt.Println("--------\t ----------\t ------------------\t -----------")
	wg := sync.WaitGroup{}
	sem := make(chan struct{}, 256) // The buffer capacity determines the size of the semaphore.
	for _, v := range targets {
		sem <- struct{}{}
		wg.Add(1) //add one unit to waitgroup for each target identified
		go func(ip string) {
			defer func() {
				<-sem     // when the channel is full, removing a struct{}{} from the semaphore channel allow a new goroutine to be started
				wg.Done() // decrements wg counter
			}()
			l, r := testConnection(ip)
			if l <= 99 { //filter out down clients
				hostname, _ := net.LookupAddr(ip) //attempt hostname lookup
				if len(hostname) == 0 {
					hostname = append(hostname, "N/A\t")
				}
				m := arp.Search(ip) // get MAC address
				if m == "" {
					m = "N/A\t\t"
				}
				split := strings.Split(ip, ".")
				lastOctet := split[3]
				lo, err := strconv.Atoi(lastOctet)
				if err != nil {
					fmt.Println(err)
				}
				h := Host{IP: ip, Hostname: hostname[0], MAC: m, RTT: r, Loss: l, LastOctet: lo}
				hosts = append(hosts, h)
			}
		}(v)
	}
	wg.Wait() // will block until wg counter reaches zero
	total := len(targets)
	//display output
	sortedHosts := sortHosts(hosts)
	for _, v := range sortedHosts {
		fmt.Println(v.Hostname, "\t", v.IP, "\t", v.MAC, "\t", v.RTT)
	}

	hostsAvailable := len(hosts)
	t1 := time.Now()
	elapsed := t1.Sub(t0)
	d, _ := time.ParseDuration("1us")
	fmt.Printf("\nTargets Scanned: %v\tUp: %v\nTotal Scan Time: %v\n\n", total, hostsAvailable, elapsed.Round(d))
}

/*
sortHosts takes a slice of hosts and sorts it by IP address
*/
func sortHosts(h []Host) []Host {
	sort.SliceStable(h, func(i, j int) bool {
		return h[i].LastOctet < h[j].LastOctet
	})
	return h
}

/*
testConnection will take a target ipv4 address and return packetloss, rtt
This function can be used to quickly verify a host is responding to ICMP packets
*/
func testConnection(target string) (float64, time.Duration) {
	pinger, err := ping.NewPinger(target)
	if err != nil {
		log.Println(err)
		return 0, time.Duration(0)
	}
	timeout := time.Millisecond * 750
	pinger.Timeout = timeout
	err = pinger.Run() // blocks until finished
	if err != nil {
		log.Println(err)
		return 0, time.Duration(0)
	}
	stats := pinger.Statistics() // get send/receive/rtt stats
	d, _ := time.ParseDuration("0.01ms")
	return stats.PacketLoss, stats.AvgRtt.Round(d)
}

/*
getTargets takes an ip address and returns a slice of ip address strings and a subnet string. the subnet /24 of the given IP
*/
func getTargets(ip string) ([]string, string) {
	split := strings.Split(ip, ".")
	var iprange []string
	subnet := split[0] + "." + split[1] + "." + split[2] + "." + "0/24"
	for i := 1; i < 255; i++ {
		fourthoctet := strconv.Itoa(i)
		target := split[0] + "." + split[1] + "." + split[2] + "." + fourthoctet
		iprange = append(iprange, target)
	}
	return iprange, subnet
}

/*
getIPv4 returns an IPv4 address as a string
*/
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

func isValidIpv4(ip string) bool {
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
