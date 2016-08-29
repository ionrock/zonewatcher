package zonewatcher

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func dnsServerAvailable(addr string) bool {
	d := DNSClient{Ns: addr}
	_, err := d.State("zone_created.com.")
	if err != nil {
		return false
	}

	return true
}

func DNSHandler() string {
	s := DnsHandler{}

	host := "127.0.0.1"
	port := "9999"
	addr := fmt.Sprintf("%s:%s", host, port)

	if dnsServerAvailable(addr) {
		return addr
	}

	s.TcpServer = s.NewServer("tcp", host, port)
	s.UdpServer = s.NewServer("udp", host, port)

	go s.Serve(s.TcpServer)
	go s.Serve(s.UdpServer)

	for {
		if dnsServerAvailable(addr) {
			break
		}
		fmt.Println("Couldn't connect to ", addr)
		time.Sleep(1 * time.Second)
	}

	return addr
}

func TestExists(t *testing.T) {
	addr := DNSHandler()

	d := DNSClient{Ns: addr}
	s, err := d.State("zone_deleted.com.")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, ZONE_DELETED, s, "zone_deleted.com should be deleted")
}

func TestNotExists(t *testing.T) {
	addr := DNSHandler()

	d := DNSClient{Ns: addr}
	s, err := d.State("zone_created.com.")
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, ZONE_CREATED, s, "zone_created.com should be created")
}
