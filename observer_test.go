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

func DNSHandler() (string, *DnsHandler) {
	backend := NewFileDnsBackend("tmp_zones")
	s := DnsHandler{
		Zones: backend,
	}

	host := "127.0.0.1"
	port := "9999"
	addr := fmt.Sprintf("%s:%s", host, port)

	if dnsServerAvailable(addr) {
		return addr, &s
	}

	go s.Serve(s.NewServer("tcp", host, port))
	go s.Serve(s.NewServer("udp", host, port))

	for {
		if dnsServerAvailable(addr) {
			break
		}
		fmt.Println("Couldn't connect to ", addr)
		time.Sleep(1 * time.Second)
	}

	return addr, &s
}

func TestZoneDeleted(t *testing.T) {
	addr, handler := DNSHandler()
	defer handler.Shutdown()

	zone := "zone_deleted.com."

	d := DNSClient{Ns: addr}
	s, err := d.State(zone)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, ZONE_DELETED, s, "zone_deleted.com should be deleted")
}

func TestZoneCreated(t *testing.T) {
	addr, handler := DNSHandler()
	defer handler.Shutdown()

	zone := "zone_created.com."

	handler.Zones.Add(zone)

	d := DNSClient{Ns: addr}
	s, err := d.State(zone)
	if err != nil {
		t.Error(err)
	}

	assert.Equal(t, ZONE_CREATED, s, "zone_created.com should be created")
}

func TestObserverWatch(t *testing.T) {
	addr, handler := DNSHandler()
	defer handler.Shutdown()

	zone := "example.com."

	d := DNSClient{Ns: addr}
	o := Observer{
		Zone:  zone,
		Ns:    addr,
		State: ZONE_CREATED,
	}
	go o.Watch(d, nil)
	handler.Zones.Add(zone)
	o.Wait()

	assert.Equal(t, STATUS_FINISHED, o.Status)
	assert.NotEmpty(t, o.Finish)

}
