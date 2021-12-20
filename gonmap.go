package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-ping/ping"
	"github.com/mostlygeek/arp"
)

func main() {
	t0 := time.Now()
	fmt.Printf("gonmap - lightweight network scanner utility\n\n")
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
	targets := getTargets(ipv4address)
	hostsAvailable := 0 // This variable is closed by the anon goroutine.  It will result in a data race without some form of mutual exclusion.
	mu := sync.Mutex{}  // Write mutex to lock access to the closed counter variable
	wg := sync.WaitGroup{}
	sem := make(chan struct{}, 256) // The buffer capacity determines the size of the semaphore.
	for _, v := range targets {     // The range loop will continue to start new goroutines until the channel is full
		sem <- struct{}{} // When the channel is full, execution will block here until a struct is read out of the channel (bounded parallelism)
		wg.Add(1)         // increments wg counter
		go func(ip string) {
			defer func() {
				<-sem     // when the channel is full, removing a struct{}{} from the semaphore channel allow a new goroutine to be started
				wg.Done() // decrements wg counter
			}()
			l, r := testConnection(ip)
			if l <= 99 { //filter out down clients
				host, _ := net.LookupAddr(ip) //attempt hostname lookup
				if len(host) == 0 {
					host = append(host, "N/A\t")
				}
				m := arp.Search(ip) // get MAC address
				if m == "" {
					m = "N/A\t\t"
				}
				fmt.Println("Host: ", ip, "\tRTT: ", r, "\tMAC:\t", m, "\tHostname:", host[0])
				mu.Lock()         // The mutex locks access to the counter variable preventing simultaneous writes to the same location in memory (data race).
				defer mu.Unlock() // unlock the mutex when it is safe to write to this memory location again.
				hostsAvailable++  // This counter is a perfect example of a data race.  Multiple goroutines could potentially write to it simultaneously
			}

		}(v)
	}
	wg.Wait() // will block until wg counter reaches zero
	total := len(targets)
	t1 := time.Now()
	elapsed := t1.Sub(t0)
	d, _ := time.ParseDuration("1us")
	fmt.Printf("\nTargets Scanned: %v\tUp: %v\nTotal Scan Time: %v\n\n", total, hostsAvailable, elapsed.Round(d))
}

/*
testConnection will take a target ipv4 address and return packetloss, rtt
This function can be used to quickly verify a host is responding to ICMP packets
*/
func testConnection(target string) (float64, time.Duration) {
	timeout := time.Millisecond * 100
	pinger, err := ping.NewPinger(target)
	if err != nil {
		log.Println(err)
		return 0, time.Duration(0)
	}
	pinger.Timeout = timeout
	pinger.Count = 1
	err = pinger.Run() // blocks until finished
	if err != nil {
		log.Println(err)
		return 0, time.Duration(0)
	}
	stats := pinger.Statistics() // get send/receive/rtt stats
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
