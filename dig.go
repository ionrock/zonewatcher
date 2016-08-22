package zonewatcher

import (
	"errors"
	"fmt"
	"log"
	"net"

	"github.com/miekg/dns"
)

type Dig interface {
	State(string) (string, error)
}

type DNSClient struct {
	SourceAddr string
	Path       string
	Ns         string
}

func (d DNSClient) Source() *net.TCPAddr {
	return &net.TCPAddr{IP: net.ParseIP(d.SourceAddr)}
}

func (d DNSClient) State(zone string) (string, error) {
	m := new(dns.Msg)
	m.SetQuestion(zone, dns.TypeSOA)

	var r *dns.Msg
	var err error

	if d.SourceAddr != "" {
		dialer := net.Dialer{LocalAddr: d.Source()}
		c, err := dialer.Dial("tcp", d.Ns)
		if err != nil {
			errmsg := fmt.Sprintf("QUERY ERROR : problem dialing %s", d.Ns)
			log.Print(errmsg)
			return "", errors.New(errmsg)
		}

		co := &dns.Conn{Conn: c}
		co.WriteMsg(m)
		r, err = co.ReadMsg()
		if err != nil {
			errmsg := fmt.Sprintf("QUERY ERROR : problem querying %s", d.Ns)
			log.Print(errmsg)
			return "", errors.New(errmsg)
		}
		co.Close()
	} else {
		c := dns.Client{}
		r, _, err = c.Exchange(m, d.Ns)
		if err != nil {
			return "", err
		}
	}

	if len(r.Answer) > 0 {
		return ZONE_CREATED, nil
	}

	return ZONE_DELETED, nil
}
